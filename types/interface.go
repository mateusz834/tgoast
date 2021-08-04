// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"go/ast"
	"go/internal/typeparams"
	"go/token"
)

// ----------------------------------------------------------------------------
// API

// An Interface represents an interface type.
type Interface struct {
	obj       *TypeName    // type name object defining this interface; or nil (for better error messages)
	methods   []*Func      // ordered list of explicitly declared methods
	embeddeds []Type       // ordered list of explicitly embedded elements
	embedPos  *[]token.Pos // positions of embedded elements; or nil (for error messages) - use pointer to save space
	complete  bool         // indicates that obj, methods, and embeddeds are set and type set can be computed

	tset *TypeSet // type set described by this interface, computed lazily
}

// typeSet returns the type set for interface t.
func (t *Interface) typeSet() *TypeSet { return computeTypeSet(nil, token.NoPos, t) }

// is reports whether interface t represents types that all satisfy f.
func (t *Interface) is(f func(Type, bool) bool) bool {
	switch t := t.typeSet().types.(type) {
	case nil, *top:
		// TODO(gri) should settle on top or nil to represent this case
		return false // we must have at least one type! (was bug)
	case *Union:
		return t.is(func(t *term) bool { return f(t.typ, t.tilde) })
	default:
		return f(t, false)
	}
}

// emptyInterface represents the empty (completed) interface
var emptyInterface = Interface{complete: true, tset: &topTypeSet}

// NewInterface returns a new interface for the given methods and embedded types.
// NewInterface takes ownership of the provided methods and may modify their types
// by setting missing receivers.
//
// Deprecated: Use NewInterfaceType instead which allows arbitrary embedded types.
func NewInterface(methods []*Func, embeddeds []*Named) *Interface {
	tnames := make([]Type, len(embeddeds))
	for i, t := range embeddeds {
		tnames[i] = t
	}
	return NewInterfaceType(methods, tnames)
}

// NewInterfaceType returns a new interface for the given methods and embedded types.
// NewInterfaceType takes ownership of the provided methods and may modify their types
// by setting missing receivers.
func NewInterfaceType(methods []*Func, embeddeds []Type) *Interface {
	if len(methods) == 0 && len(embeddeds) == 0 {
		return &emptyInterface
	}

	// set method receivers if necessary
	typ := new(Interface)
	for _, m := range methods {
		if sig := m.typ.(*Signature); sig.recv == nil {
			sig.recv = NewVar(m.pos, m.pkg, "", typ)
		}
	}

	// sort for API stability
	sortMethods(methods)

	typ.methods = methods
	typ.embeddeds = embeddeds
	typ.complete = true

	return typ
}

// NumExplicitMethods returns the number of explicitly declared methods of interface t.
func (t *Interface) NumExplicitMethods() int { return len(t.methods) }

// ExplicitMethod returns the i'th explicitly declared method of interface t for 0 <= i < t.NumExplicitMethods().
// The methods are ordered by their unique Id.
func (t *Interface) ExplicitMethod(i int) *Func { return t.methods[i] }

// NumEmbeddeds returns the number of embedded types in interface t.
func (t *Interface) NumEmbeddeds() int { return len(t.embeddeds) }

// Embedded returns the i'th embedded defined (*Named) type of interface t for 0 <= i < t.NumEmbeddeds().
// The result is nil if the i'th embedded type is not a defined type.
//
// Deprecated: Use EmbeddedType which is not restricted to defined (*Named) types.
func (t *Interface) Embedded(i int) *Named { tname, _ := t.embeddeds[i].(*Named); return tname }

// EmbeddedType returns the i'th embedded type of interface t for 0 <= i < t.NumEmbeddeds().
func (t *Interface) EmbeddedType(i int) Type { return t.embeddeds[i] }

// NumMethods returns the total number of methods of interface t.
func (t *Interface) NumMethods() int { return t.typeSet().NumMethods() }

// Method returns the i'th method of interface t for 0 <= i < t.NumMethods().
// The methods are ordered by their unique Id.
func (t *Interface) Method(i int) *Func { return t.typeSet().Method(i) }

// Empty reports whether t is the empty interface.
func (t *Interface) Empty() bool { return t.typeSet().IsTop() }

// IsComparable reports whether each type in interface t's type set is comparable.
func (t *Interface) IsComparable() bool { return t.typeSet().IsComparable() }

// IsConstraint reports whether interface t is not just a method set.
func (t *Interface) IsConstraint() bool { return !t.typeSet().IsMethodSet() }

// isSatisfiedBy reports whether interface t's type list is satisfied by the type typ.
// If the type list is empty (absent), typ trivially satisfies the interface.
// TODO(gri) This is not a great name. Eventually, we should have a more comprehensive
//           "implements" predicate.
func (t *Interface) isSatisfiedBy(typ Type) bool {
	t.Complete()
	switch t := t.typeSet().types.(type) {
	case nil:
		return true // no type restrictions
	case *Union:
		r, _ := t.intersect(typ, false)
		return r != nil
	default:
		return Identical(t, typ)
	}
}

// Complete computes the interface's type set. It must be called by users of
// NewInterfaceType and NewInterface after the interface's embedded types are
// fully defined and before using the interface type in any way other than to
// form other types. The interface must not contain duplicate methods or a
// panic occurs. Complete returns the receiver.
//
// Deprecated: Type sets are now computed lazily, on demand; this function
//             is only here for backward-compatibility. It does not have to
//             be called explicitly anymore.
func (t *Interface) Complete() *Interface {
	// Some tests are still depending on the state change
	// (string representation of an Interface not containing an
	// /* incomplete */ marker) caused by the explicit Complete
	// call, so we compute the type set eagerly here.
	t.complete = true
	t.typeSet()
	return t
}

func (t *Interface) Underlying() Type { return t }
func (t *Interface) String() string   { return TypeString(t, nil) }

// ----------------------------------------------------------------------------
// Implementation

func (check *Checker) interfaceType(ityp *Interface, iface *ast.InterfaceType, def *Named) {
	var tlist []ast.Expr
	var tname *ast.Ident // "type" name of first entry in a type list declaration

	addEmbedded := func(pos token.Pos, typ Type) {
		ityp.embeddeds = append(ityp.embeddeds, typ)
		if ityp.embedPos == nil {
			ityp.embedPos = new([]token.Pos)
		}
		*ityp.embedPos = append(*ityp.embedPos, pos)
	}

	for _, f := range iface.Methods.List {
		if len(f.Names) == 0 {
			// We have an embedded type; possibly a union of types.
			addEmbedded(f.Type.Pos(), parseUnion(check, flattenUnion(nil, f.Type)))
			continue
		}

		// We have a method with name f.Names[0], or a type
		// of a type list (name.Name == "type").
		// (The parser ensures that there's only one method
		// and we don't care if a constructed AST has more.)
		name := f.Names[0]
		if name.Name == "_" {
			check.errorf(name, _BlankIfaceMethod, "invalid method name _")
			continue // ignore
		}

		// TODO(rfindley) Remove type list handling once the parser doesn't accept type lists anymore.
		if name.Name == "type" {
			// Report an error for the first type list per interface
			// if we don't allow type lists, but continue.
			if !allowTypeLists && tlist == nil {
				check.softErrorf(name, _Todo, "use generalized embedding syntax instead of a type list")
			}
			// For now, collect all type list entries as if it
			// were a single union, where each union element is
			// of the form ~T.
			// TODO(rfindley) remove once we disallow type lists
			op := new(ast.UnaryExpr)
			op.Op = token.TILDE
			op.X = f.Type
			tlist = append(tlist, op)
			// Report an error if we have multiple type lists in an
			// interface, but only if they are permitted in the first place.
			if allowTypeLists && tname != nil && tname != name {
				check.errorf(name, _Todo, "cannot have multiple type lists in an interface")
			}
			tname = name
			continue
		}

		typ := check.typ(f.Type)
		sig, _ := typ.(*Signature)
		if sig == nil {
			if typ != Typ[Invalid] {
				check.invalidAST(f.Type, "%s is not a method signature", typ)
			}
			continue // ignore
		}

		// Always type-check method type parameters but complain if they are not enabled.
		// (This extra check is needed here because interface method signatures don't have
		// a receiver specification.)
		if sig.tparams != nil {
			var at positioner = f.Type
			if tparams := typeparams.Get(f.Type); tparams != nil {
				at = tparams
			}
			check.errorf(at, _Todo, "methods cannot have type parameters")
		}

		// use named receiver type if available (for better error messages)
		var recvTyp Type = ityp
		if def != nil {
			recvTyp = def
		}
		sig.recv = NewVar(name.Pos(), check.pkg, "", recvTyp)

		m := NewFunc(name.Pos(), check.pkg, name.Name, sig)
		check.recordDef(name, m)
		ityp.methods = append(ityp.methods, m)
	}

	// type constraints
	if tlist != nil {
		// TODO(rfindley): this differs from types2 due to the use of Pos() below,
		// which should actually be on the ~. Confirm that this position is correct.
		addEmbedded(tlist[0].Pos(), parseUnion(check, tlist))
	}

	// All methods and embedded elements for this interface are collected;
	// i.e., this interface is may be used in a type set computation.
	ityp.complete = true

	if len(ityp.methods) == 0 && len(ityp.embeddeds) == 0 {
		// empty interface
		ityp.tset = &topTypeSet
		return
	}

	// sort for API stability
	sortMethods(ityp.methods)
	// (don't sort embeddeds: they must correspond to *embedPos entries)

	// Compute type set with a non-nil *Checker as soon as possible
	// to report any errors. Subsequent uses of type sets will use
	// this computed type set and won't need to pass in a *Checker.
	check.later(func() { computeTypeSet(check, iface.Pos(), ityp) })
}

func flattenUnion(list []ast.Expr, x ast.Expr) []ast.Expr {
	if o, _ := x.(*ast.BinaryExpr); o != nil && o.Op == token.OR {
		list = flattenUnion(list, o.X)
		x = o.Y
	}
	return append(list, x)
}
