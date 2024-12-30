// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements a custom generator to create various go/types
// source files from the corresponding types2 files.

package types_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/format"
	"github.com/mateusz834/tgoast/internal/diff"
	"github.com/mateusz834/tgoast/parser"
	"github.com/mateusz834/tgoast/token"
)

var filesToWrite = flag.String("write", "", `go/types files to generate, or "all" for all files`)

const (
	srcDir = "/src/cmd/compile/internal/types2/"
	dstDir = "/src/go/types/"
)

// TestGenerate verifies that generated files in go/types match their types2
// counterpart. If -write is set, this test actually writes the expected
// content to go/types; otherwise, it just compares with the existing content.
func TestGenerate(t *testing.T) {
	// If filesToWrite is set, write the generated content to disk.
	// In the special case of "all", write all files in filemap.
	write := *filesToWrite != ""
	var files []string // files to process
	if *filesToWrite != "" && *filesToWrite != "all" {
		files = strings.Split(*filesToWrite, ",")
	} else {
		for file := range filemap {
			files = append(files, file)
		}
	}

	for _, filename := range files {
		generate(t, filename, write)
	}
}

func generate(t *testing.T, filename string, write bool) {
	// parse src (cmd/compile/internal/types2)
	srcFilename := filepath.FromSlash(runtime.GOROOT() + srcDir + filename)
	file, err := parser.ParseFile(fset, srcFilename, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// fix package name
	file.Name.Name = strings.ReplaceAll(file.Name.Name, "types2", "types")

	// rewrite AST as needed
	if action := filemap[filename]; action != nil {
		action(file)
	}

	// format AST
	var buf bytes.Buffer
	rel, _ := filepath.Rel(dstDir, srcDir)
	fmt.Fprintf(&buf, "// Code generated by \"go test -run=Generate -write=all\"; DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "// Source: %s/%s\n\n", filepath.ToSlash(rel), filename)
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatal(err)
	}
	generatedContent := buf.Bytes()

	// read dst (go/types)
	dstFilename := filepath.FromSlash(runtime.GOROOT() + dstDir + filename)
	onDiskContent, err := os.ReadFile(dstFilename)
	if err != nil {
		t.Fatalf("reading %q: %v", filename, err)
	}

	// compare on-disk dst with buffer generated from src.
	if d := diff.Diff(filename+" (on disk in "+dstDir+")", onDiskContent, filename+" (generated from "+srcDir+")", generatedContent); d != nil {
		if write {
			t.Logf("applying change:\n%s", d)
			if err := os.WriteFile(dstFilename, generatedContent, 0o644); err != nil {
				t.Fatalf("writing %q: %v", filename, err)
			}
		} else {
			t.Errorf("file on disk in %s is stale:\n%s", dstDir, d)
		}
	}
}

type action func(in *ast.File)

var filemap = map[string]action{
	"alias.go": fixTokenPos,
	"assignments.go": func(f *ast.File) {
		renameImportPath(f, `"cmd/compile/internal/syntax"->"go/ast"`)
		renameSelectorExprs(f, "syntax.Name->ast.Ident", "ident.Value->ident.Name", "ast.Pos->token.Pos") // must happen before renaming identifiers
		renameIdents(f, "syntax->ast", "poser->positioner", "nopos->noposn")
	},
	"array.go":          nil,
	"api_predicates.go": nil,
	"basic.go":          nil,
	"builtins.go": func(f *ast.File) {
		renameImportPath(f, `"cmd/compile/internal/syntax"->"go/ast"`)
		renameIdents(f, "syntax->ast")
		renameSelectors(f, "ArgList->Args")
		fixSelValue(f)
		fixAtPosCall(f)
	},
	"builtins_test.go": func(f *ast.File) {
		renameImportPath(f, `"cmd/compile/internal/syntax"->"go/ast"`, `"cmd/compile/internal/types2"->"go/types"`)
		renameSelectorExprs(f, "syntax.Name->ast.Ident", "p.Value->p.Name") // must happen before renaming identifiers
		renameIdents(f, "syntax->ast")
	},
	"chan.go":         nil,
	"const.go":        fixTokenPos,
	"context.go":      nil,
	"context_test.go": nil,
	"conversions.go":  nil,
	"errors_test.go":  func(f *ast.File) { renameIdents(f, "nopos->noposn") },
	"errsupport.go":   nil,
	"gccgosizes.go":   nil,
	"gcsizes.go":      func(f *ast.File) { renameIdents(f, "IsSyncAtomicAlign64->_IsSyncAtomicAlign64") },
	"hilbert_test.go": func(f *ast.File) { renameImportPath(f, `"cmd/compile/internal/types2"->"go/types"`) },
	"infer.go":        func(f *ast.File) { fixTokenPos(f); fixInferSig(f) },
	"initorder.go":    nil,
	// "initorder.go": fixErrErrorfCall, // disabled for now due to unresolved error_ use implications for gopls
	"instantiate.go":      func(f *ast.File) { fixTokenPos(f); fixCheckErrorfCall(f) },
	"instantiate_test.go": func(f *ast.File) { renameImportPath(f, `"cmd/compile/internal/types2"->"go/types"`) },
	"lookup.go":           func(f *ast.File) { fixTokenPos(f) },
	"main_test.go":        nil,
	"map.go":              nil,
	"mono.go": func(f *ast.File) {
		fixTokenPos(f)
		insertImportPath(f, `"go/ast"`)
		renameSelectorExprs(f, "syntax.Expr->ast.Expr")
	},
	"named.go":  func(f *ast.File) { fixTokenPos(f); renameSelectors(f, "Trace->_Trace") },
	"object.go": func(f *ast.File) { fixTokenPos(f); renameIdents(f, "NewTypeNameLazy->_NewTypeNameLazy") },
	// TODO(gri) needs adjustments for TestObjectString - disabled for now
	// "object_test.go": func(f *ast.File) { renameImportPath(f, `"cmd/compile/internal/types2"->"go/types"`) },
	"objset.go": nil,
	"operand.go": func(f *ast.File) {
		insertImportPath(f, `"go/token"`)
		renameImportPath(f, `"cmd/compile/internal/syntax"->"go/ast"`)
		renameSelectorExprs(f,
			"syntax.Pos->token.Pos", "syntax.LitKind->token.Token",
			"syntax.IntLit->token.INT", "syntax.FloatLit->token.FLOAT",
			"syntax.ImagLit->token.IMAG", "syntax.RuneLit->token.CHAR",
			"syntax.StringLit->token.STRING") // must happen before renaming identifiers
		renameIdents(f, "syntax->ast")
	},
	"package.go":       nil,
	"pointer.go":       nil,
	"predicates.go":    nil,
	"scope.go":         func(f *ast.File) { fixTokenPos(f); renameIdents(f, "Squash->squash", "InsertLazy->_InsertLazy") },
	"selection.go":     nil,
	"sizes.go":         func(f *ast.File) { renameIdents(f, "IsSyncAtomicAlign64->_IsSyncAtomicAlign64") },
	"slice.go":         nil,
	"subst.go":         func(f *ast.File) { fixTokenPos(f); renameSelectors(f, "Trace->_Trace") },
	"termlist.go":      nil,
	"termlist_test.go": nil,
	"tuple.go":         nil,
	"typelists.go":     nil,
	"typeset.go":       func(f *ast.File) { fixTokenPos(f); renameSelectors(f, "Trace->_Trace") },
	"typeparam.go":     nil,
	"typeterm_test.go": nil,
	"typeterm.go":      nil,
	"typestring.go":    nil,
	"under.go":         nil,
	"unify.go":         fixSprintf,
	"universe.go":      fixGlobalTypVarDecl,
	"util_test.go":     fixTokenPos,
	"validtype.go":     func(f *ast.File) { fixTokenPos(f); renameSelectors(f, "Trace->_Trace") },
}

// TODO(gri) We should be able to make these rewriters more configurable/composable.
//           For now this is a good starting point.

// A renameMap maps old strings to new strings.
type renameMap map[string]string

// makeRenameMap returns a renameMap populates from renames entries of the form "from->to".
func makeRenameMap(renames ...string) renameMap {
	m := make(renameMap)
	for _, r := range renames {
		s := strings.Split(r, "->")
		if len(s) != 2 {
			panic("invalid rename entry: " + r)
		}
		m[s[0]] = s[1]
	}
	return m
}

// rename renames the given string s if a corresponding rename exists in m.
func (m renameMap) rename(s *string) {
	if r, ok := m[*s]; ok {
		*s = r
	}
}

// renameSel renames a selector expression of the form a.x to b.x (where a, b are identifiers)
// if m contains the ("a.x" : "b.y") key-value pair.
func (m renameMap) renameSel(n *ast.SelectorExpr) {
	if a, _ := n.X.(*ast.Ident); a != nil {
		a_x := a.Name + "." + n.Sel.Name
		if r, ok := m[a_x]; ok {
			b_y := strings.Split(r, ".")
			if len(b_y) != 2 {
				panic("invalid selector expression: " + r)
			}
			a.Name = b_y[0]
			n.Sel.Name = b_y[1]
		}
	}
}

// renameIdents renames identifiers: each renames entry is of the form "from->to".
// Note: This doesn't change the use of the identifiers in comments.
func renameIdents(f *ast.File, renames ...string) {
	m := makeRenameMap(renames...)
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			m.rename(&n.Name)
			return false
		}
		return true
	})
}

// renameSelectors is like renameIdents but only looks at selectors.
func renameSelectors(f *ast.File, renames ...string) {
	m := makeRenameMap(renames...)
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			m.rename(&n.Sel.Name)
			return false
		}
		return true
	})

}

// renameSelectorExprs is like renameIdents but only looks at selector expressions.
// Each renames entry must be of the form "x.a->y.b".
func renameSelectorExprs(f *ast.File, renames ...string) {
	m := makeRenameMap(renames...)
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			m.renameSel(n)
			return false
		}
		return true
	})
}

// renameImportPath is like renameIdents but renames import paths.
func renameImportPath(f *ast.File, renames ...string) {
	m := makeRenameMap(renames...)
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			if n.Path.Kind != token.STRING {
				panic("invalid import path")
			}
			m.rename(&n.Path.Value)
			return false
		}
		return true
	})
}

// insertImportPath inserts the given import path.
// There must be at least one import declaration present already.
func insertImportPath(f *ast.File, path string) {
	for _, d := range f.Decls {
		if g, _ := d.(*ast.GenDecl); g != nil && g.Tok == token.IMPORT {
			g.Specs = append(g.Specs, &ast.ImportSpec{Path: &ast.BasicLit{ValuePos: g.End(), Kind: token.STRING, Value: path}})
			return
		}
	}
	panic("no import declaration present")
}

// fixTokenPos changes imports of "cmd/compile/internal/syntax" to "go/token",
// uses of syntax.Pos to token.Pos, and calls to x.IsKnown() to x.IsValid().
func fixTokenPos(f *ast.File) {
	m := makeRenameMap(`"cmd/compile/internal/syntax"->"go/token"`, "syntax.Pos->token.Pos", "IsKnown->IsValid")
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			// rewrite import path "cmd/compile/internal/syntax" to "go/token"
			if n.Path.Kind != token.STRING {
				panic("invalid import path")
			}
			m.rename(&n.Path.Value)
			return false
		case *ast.SelectorExpr:
			// rewrite syntax.Pos to token.Pos
			m.renameSel(n)
		case *ast.CallExpr:
			// rewrite x.IsKnown() to x.IsValid()
			if fun, _ := n.Fun.(*ast.SelectorExpr); fun != nil && len(n.Args) == 0 {
				m.rename(&fun.Sel.Name)
				return false
			}
		}
		return true
	})
}

// fixSelValue updates the selector x.Sel.Value to x.Sel.Name.
func fixSelValue(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if n.Sel.Name == "Value" {
				if selx, _ := n.X.(*ast.SelectorExpr); selx != nil && selx.Sel.Name == "Sel" {
					n.Sel.Name = "Name"
					return false
				}
			}
		}
		return true
	})
}

// fixInferSig updates the Checker.infer signature to use a positioner instead of a token.Position
// as first argument, renames the argument from "pos" to "posn", and updates a few internal uses of
// "pos" to "posn" and "posn.Pos()" respectively.
func fixInferSig(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Name.Name == "infer" {
				// rewrite (pos token.Pos, ...) to (posn positioner, ...)
				par := n.Type.Params.List[0]
				if len(par.Names) == 1 && par.Names[0].Name == "pos" {
					par.Names[0] = newIdent(par.Names[0].Pos(), "posn")
					par.Type = newIdent(par.Type.Pos(), "positioner")
					return true
				}
			}
		case *ast.CallExpr:
			if selx, _ := n.Fun.(*ast.SelectorExpr); selx != nil {
				switch selx.Sel.Name {
				case "renameTParams":
					// rewrite check.renameTParams(pos, ... ) to check.renameTParams(posn.Pos(), ... )
					if isIdent(n.Args[0], "pos") {
						pos := n.Args[0].Pos()
						fun := &ast.SelectorExpr{X: newIdent(pos, "posn"), Sel: newIdent(pos, "Pos")}
						arg := &ast.CallExpr{Fun: fun, Lparen: pos, Args: nil, Ellipsis: token.NoPos, Rparen: pos}
						n.Args[0] = arg
						return false
					}
				case "addf":
					// rewrite err.addf(pos, ...) to err.addf(posn, ...)
					if isIdent(n.Args[0], "pos") {
						pos := n.Args[0].Pos()
						arg := newIdent(pos, "posn")
						n.Args[0] = arg
						return false
					}
				case "allowVersion":
					// rewrite check.allowVersion(pos, ...) to check.allowVersion(posn, ...)
					if isIdent(n.Args[0], "pos") {
						pos := n.Args[0].Pos()
						arg := newIdent(pos, "posn")
						n.Args[0] = arg
						return false
					}
				}
			}
		}
		return true
	})
}

// fixAtPosCall updates calls of the form atPos(x) to x.Pos() in argument lists of (check).dump calls.
// TODO(gri) can we avoid this and just use atPos consistently in go/types and types2?
func fixAtPosCall(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if selx, _ := n.Fun.(*ast.SelectorExpr); selx != nil && selx.Sel.Name == "dump" {
				for i, arg := range n.Args {
					if call, _ := arg.(*ast.CallExpr); call != nil {
						// rewrite xxx.dump(..., atPos(x), ...) to xxx.dump(..., x.Pos(), ...)
						if isIdent(call.Fun, "atPos") {
							pos := call.Args[0].Pos()
							fun := &ast.SelectorExpr{X: call.Args[0], Sel: newIdent(pos, "Pos")}
							n.Args[i] = &ast.CallExpr{Fun: fun, Lparen: pos, Rparen: pos}
							return false
						}
					}
				}
			}
		}
		return true
	})
}

// fixErrErrorfCall updates calls of the form err.addf(obj, ...) to err.addf(obj.Pos(), ...).
func fixErrErrorfCall(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if selx, _ := n.Fun.(*ast.SelectorExpr); selx != nil {
				if isIdent(selx.X, "err") {
					switch selx.Sel.Name {
					case "errorf":
						// rewrite err.addf(obj, ... ) to err.addf(obj.Pos(), ... )
						if ident, _ := n.Args[0].(*ast.Ident); ident != nil && ident.Name == "obj" {
							pos := n.Args[0].Pos()
							fun := &ast.SelectorExpr{X: ident, Sel: newIdent(pos, "Pos")}
							n.Args[0] = &ast.CallExpr{Fun: fun, Lparen: pos, Rparen: pos}
							return false
						}
					}
				}
			}
		}
		return true
	})
}

// fixCheckErrorfCall updates calls of the form check.errorf(pos, ...) to check.errorf(atPos(pos), ...).
func fixCheckErrorfCall(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if selx, _ := n.Fun.(*ast.SelectorExpr); selx != nil {
				if isIdent(selx.X, "check") {
					switch selx.Sel.Name {
					case "errorf":
						// rewrite check.errorf(pos, ... ) to check.errorf(atPos(pos), ... )
						if ident := asIdent(n.Args[0], "pos"); ident != nil {
							pos := n.Args[0].Pos()
							fun := newIdent(pos, "atPos")
							n.Args[0] = &ast.CallExpr{Fun: fun, Lparen: pos, Args: []ast.Expr{ident}, Rparen: pos}
							return false
						}
					}
				}
			}
		}
		return true
	})
}

// fixGlobalTypVarDecl changes the global Typ variable from an array to a slice
// (in types2 we use an array for efficiency, in go/types it's a slice and we
// cannot change that).
func fixGlobalTypVarDecl(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			// rewrite type Typ = [...]Type{...} to type Typ = []Type{...}
			if len(n.Names) == 1 && n.Names[0].Name == "Typ" && len(n.Values) == 1 {
				n.Values[0].(*ast.CompositeLit).Type.(*ast.ArrayType).Len = nil
				return false
			}
		}
		return true
	})
}

// fixSprintf adds an extra nil argument for the *token.FileSet parameter in sprintf calls.
func fixSprintf(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if isIdent(n.Fun, "sprintf") && len(n.Args) >= 4 /* ... args */ {
				n.Args = insert(n.Args, 1, newIdent(n.Args[1].Pos(), "nil"))
				return false
			}
		}
		return true
	})
}

// asIdent returns x as *ast.Ident if it is an identifier with the given name.
func asIdent(x ast.Node, name string) *ast.Ident {
	if ident, _ := x.(*ast.Ident); ident != nil && ident.Name == name {
		return ident
	}
	return nil
}

// isIdent reports whether x is an identifier with the given name.
func isIdent(x ast.Node, name string) bool {
	return asIdent(x, name) != nil
}

// newIdent returns a new identifier with the given position and name.
func newIdent(pos token.Pos, name string) *ast.Ident {
	id := ast.NewIdent(name)
	id.NamePos = pos
	return id
}

// insert inserts x at list[at] and moves the remaining elements up.
func insert(list []ast.Expr, at int, x ast.Expr) []ast.Expr {
	list = append(list, nil)
	copy(list[at+1:], list[at:])
	list[at] = x
	return list
}
