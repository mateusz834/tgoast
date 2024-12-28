package types_test

import (
	"maps"
	"sync"
	"testing"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/importer"
	"github.com/mateusz834/tgoast/parser"
	"github.com/mateusz834/tgoast/token"
	. "github.com/mateusz834/tgoast/types"
)

// TODO: only ".tgo" files might contain "tgo nodes".
// maybe that should be part of the parser? With a flag parser.AllowTgo,
// this probably should also propagate to the scanner scanner.AllowTgo
// so, that Tgo is opt-in, and there wouldn't be any differences in errors
// or the AST for code that uses our fork, but witout opting in.
// and fuzz whether in non-tgo mode, the forked *non-tgo" modoe parser
// prooduces the same AST and errors and same thing with the scanner.
// But this fuzz, needs to only run on the same version of Go as the fork.
// So a runtime.GoVersion check, but for CI it should fail in that case, so we
// know when we are not running it?

// TODO: better support for return's, we should allow returing non-nils from
// whithin inside of a tag. But this (not nil) is a runtime property, wheter something is nil
// we should have a check that checks wheter error is indeed not nil, if nil the we return
// an error, that notes that (or panic?)
//
//var ErrUnclosedTag = errors.New("unclosed tag")
//
//func ErrorReturn(err error) error {
//	if err == nil {
//		return ErrUnclosedTag
//  }
//	return err
//}
// and a vet? that detects such return nil? We are not able to detect that in transpiler.
// Nil can be:
// const nil = true
// somwhere else (in a different file).

// TODO: Doctype plus some "type-checking" for it (so that it must be first?). Think about that.
// TODO: HTML comments (what if someone wants to render
// something dynamicaly in a comment :), do we want to support that somehow?)
//	<!--"comment"-> Quotes???

// TODO: script/style tags (and figure out whether there are other tags that need this).
// in those we need to only accept html.UnsafeHTML in template literals?
// Even without any other string "sth\{html.UnsafeHTML("test")}", the problem also is
// that "\{"<div>"}" is going too work fine (no explicit conversion required).

func TestTgoTest(t *testing.T) {
	const src = `package test

import "github.com/mateusz834/tgo"

func _(tgo.Ctx) error {
	<div>
		<div>
		</div>
	</div>
	return nil
}
`
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "test.tgo", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Error: func(err error) {
			t.Logf("err: %v\n", err)
		},
		Importer: &tgoDefaultImporter{importer.Default().(ImporterFrom)},
	}
	p, err := cfg.Check("test", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(p)
}

func TestTgo(t *testing.T) {
	testDirFiles(t, "../internal/types/testdata/tgo", false)
}

func TestTgoInfos(t *testing.T) {
	const src = `package pkg

	import "github.com/mateusz834/tgo"

func test(tgo.Ctx) error {
	<article
		a := 1
		@attr="\{a} \{"sth"}"
	>
		b := "str"
		"\{b}"
		<div
			c := 3
			panic(c)
		>
			d := 4
			"\{d}"
			"\{b} \{d}"
			const sth = "a"
			"\{1} \{1+2} \{"a"+"b"} \{sth}"
		</div>
	</article>
	return nil
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "pkg.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	infos := Info{
		Types:      map[ast.Expr]TypeAndValue{},
		Instances:  map[*ast.Ident]Instance{},
		Defs:       map[*ast.Ident]Object{},
		Uses:       map[*ast.Ident]Object{},
		Selections: map[*ast.SelectorExpr]*Selection{},
		Scopes:     map[ast.Node]*Scope{},
	}

	cfg := Config{Importer: &tgoDefaultImporter{importer.Default().(ImporterFrom)}}
	pkg, err := cfg.Check("pkg", fset, []*ast.File{f}, &infos)
	if err != nil {
		t.Fatal(err)
	}

	fun := f.Decls[1].(*ast.FuncDecl)
	article := fun.Body.List[0].(*ast.ElementBlockStmt)
	div := article.Body[2].(*ast.ElementBlockStmt)

	articleOpenTagAttrTemplateLit := article.OpenTag.Body[1].(*ast.AttributeStmt).Value.(*ast.TemplateLiteralExpr)
	articleTemplateLit := article.Body[1].(*ast.ExprStmt).X.(*ast.TemplateLiteralExpr)
	divTemplateLit1 := div.Body[1].(*ast.ExprStmt).X.(*ast.TemplateLiteralExpr)
	divTemplateLit2 := div.Body[2].(*ast.ExprStmt).X.(*ast.TemplateLiteralExpr)
	divTemplateLit3 := div.Body[4].(*ast.ExprStmt).X.(*ast.TemplateLiteralExpr)

	t.Run("types", func(t *testing.T) {
		wantTypes := map[ast.Expr]Type{
			articleOpenTagAttrTemplateLit.Parts[0].X: Typ[Int],
			articleOpenTagAttrTemplateLit.Parts[1].X: Typ[UntypedString],
			articleTemplateLit.Parts[0].X:            Typ[String],
			divTemplateLit1.Parts[0].X:               Typ[Int],
			divTemplateLit2.Parts[0].X:               Typ[String],
			divTemplateLit2.Parts[1].X:               Typ[Int],
			divTemplateLit3.Parts[0].X:               Typ[UntypedInt],
			divTemplateLit3.Parts[1].X:               Typ[UntypedInt],
			divTemplateLit3.Parts[2].X:               Typ[UntypedString],
			divTemplateLit3.Parts[3].X:               Typ[UntypedString],
		}

		for k, v := range wantTypes {
			if typ := infos.Types[k].Type; typ == nil {
				t.Errorf("missing type for: %#v", k)
			} else if v != typ {
				t.Errorf("unexpected type for: %#v; got = %v; want = %v", k, typ, v)
			}
			delete(wantTypes, k)
		}
	})

	t.Run("instances", func(t *testing.T) {
		if len(infos.Instances) != 0 {
			t.Errorf("len(info.Instances) = %v; want = 0", len(infos.Instances))
		}
	})

	aDef := article.OpenTag.Body[0].(*ast.AssignStmt).Lhs[0].(*ast.Ident)
	bDef := article.Body[0].(*ast.AssignStmt).Lhs[0].(*ast.Ident)
	cDef := div.OpenTag.Body[0].(*ast.AssignStmt).Lhs[0].(*ast.Ident)
	dDef := div.Body[0].(*ast.AssignStmt).Lhs[0].(*ast.Ident)
	sthDef := div.Body[3].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]

	t.Run("defs", func(t *testing.T) {
		wantDefs := map[*ast.Ident]bool{
			f.Name:   true,
			fun.Name: true,
			aDef:     true,
			bDef:     true,
			cDef:     true,
			dDef:     true,
			sthDef:   true,
		}

		defs := maps.Clone(infos.Defs)
		for k := range wantDefs {
			if _, ok := defs[k]; !ok {
				t.Errorf("missing def for: %#v", k)
			}
			delete(defs, k)
		}
		for k := range defs {
			t.Errorf("unexpected def for: %#v", k)
		}
	})

	t.Run("uses", func(t *testing.T) {
		wantUses := map[*ast.Ident]*ast.Ident{
			articleOpenTagAttrTemplateLit.Parts[0].X.(*ast.Ident): aDef,
			articleTemplateLit.Parts[0].X.(*ast.Ident):            bDef,
			divTemplateLit1.Parts[0].X.(*ast.Ident):               dDef,
			divTemplateLit2.Parts[0].X.(*ast.Ident):               bDef,
			divTemplateLit2.Parts[1].X.(*ast.Ident):               dDef,
			divTemplateLit3.Parts[3].X.(*ast.Ident):               sthDef,
		}

		uses := maps.Clone(infos.Uses)
		for use, def := range wantUses {
			obj := uses[use]

			var foundIdent *ast.Ident
			for gotDef, o := range infos.Defs {
				if o == obj {
					foundIdent = gotDef
					break
				}
			}

			if foundIdent != def {
				t.Errorf("ident %#v is a use of def = %#v; want = %#v", use, foundIdent, def)
			}

			delete(uses, use)
		}
	})

	t.Run("scopes", func(t *testing.T) {
		wantScopesFor := map[ast.Node]bool{
			f:               true,
			fun.Type:        true,
			article.OpenTag: true,
			article:         true,
			div.OpenTag:     true,
			div:             true,
		}

		scopes := maps.Clone(infos.Scopes)
		for s := range wantScopesFor {
			if scopes[s] == nil {
				t.Errorf("missing scope for: %#v", s)
			}
			delete(scopes, s)
		}
		for s := range scopes {
			t.Errorf("unexpected scope: %#v", s)
		}
	})

	_ = pkg
}

type tgoDefaultImporter struct {
	i ImporterFrom
}

func (f *tgoDefaultImporter) Import(path string) (*Package, error) {
	if path == "github.com/mateusz834/tgo" {
		return tgoPkg()
	}
	return f.i.Import(path)
}

func (f *tgoDefaultImporter) ImportFrom(path, dir string, mode ImportMode) (*Package, error) {
	if path == "github.com/mateusz834/tgo" {
		return tgoPkg()
	}
	return f.i.ImportFrom(path, dir, mode)
}

// TODO: test that proves (only for CI) that this is the same as in the tgo repo.
var tgoPkg = sync.OnceValues(func() (*Package, error) {
	const tgoModuleSrc = `package tgo
type Ctx struct{}
type Error = error
type UnsafeHTML string
type DynamicWriteAllowed interface {
	string|UnsafeHTML|int|uint|rune
}
func DynamicWrite[T DynamicWriteAllowed](t T) {
}
`
	fset := token.NewFileSet()
	tgoModuleFile, err := parser.ParseFile(fset, "tgo.go", tgoModuleSrc, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	tgoPkg, err := new(Config).Check("github.com/mateusz834/tgoast", fset, []*ast.File{tgoModuleFile}, nil)
	if err != nil {
		return nil, err
	}

	return tgoPkg, nil
})
