// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements a custom generator to create various go/types
// source files from the corresponding types2 files.

package types_test

import (
	"bytes"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"internal/diff"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	// parse src
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
	buf.WriteString("// Code generated by \"go test -run=Generate -write=all\"; DO NOT EDIT.\n\n")
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatal(err)
	}
	generatedContent := buf.Bytes()

	dstFilename := filepath.FromSlash(runtime.GOROOT() + dstDir + filename)
	onDiskContent, err := os.ReadFile(dstFilename)
	if err != nil {
		t.Fatalf("reading %q: %v", filename, err)
	}

	if d := diff.Diff(filename+" (on disk)", onDiskContent, filename+" (generated)", generatedContent); d != nil {
		if write {
			t.Logf("applying change:\n%s", d)
			if err := os.WriteFile(dstFilename, generatedContent, 0o644); err != nil {
				t.Fatalf("writing %q: %v", filename, err)
			}
		} else {
			t.Errorf("generated file content does not match:\n%s", string(d))
		}
	}
}

type action func(in *ast.File)

var filemap = map[string]action{
	"array.go":        nil,
	"basic.go":        nil,
	"chan.go":         nil,
	"const.go":        func(f *ast.File) { fixTokenPos(f) },
	"context.go":      nil,
	"context_test.go": nil,
	"gccgosizes.go":   nil,
	"hilbert_test.go": nil,
	"infer.go": func(f *ast.File) {
		fixTokenPos(f)
		fixInferSig(f)
	},
	// "initorder.go": fixErrErrorfCall, // disabled for now due to unresolved error_ use implications for gopls
	"instantiate.go":      func(f *ast.File) { fixTokenPos(f); fixCheckErrorfCall(f) },
	"instantiate_test.go": func(f *ast.File) { renameImportPath(f, `"cmd/compile/internal/types2"`, `"go/types"`) },
	"lookup.go":           func(f *ast.File) { fixTokenPos(f) },
	"main_test.go":        nil,
	"map.go":              nil,
	"named.go":            func(f *ast.File) { fixTokenPos(f); fixTraceSel(f) },
	"object.go":           func(f *ast.File) { fixTokenPos(f); renameIdent(f, "NewTypeNameLazy", "_NewTypeNameLazy") },
	"object_test.go":      func(f *ast.File) { renameImportPath(f, `"cmd/compile/internal/types2"`, `"go/types"`) },
	"objset.go":           nil,
	"package.go":          nil,
	"pointer.go":          nil,
	"predicates.go":       nil,
	"scope.go": func(f *ast.File) {
		fixTokenPos(f)
		renameIdent(f, "Squash", "squash")
		renameIdent(f, "InsertLazy", "_InsertLazy")
	},
	"selection.go":     nil,
	"sizes.go":         func(f *ast.File) { renameIdent(f, "IsSyncAtomicAlign64", "_IsSyncAtomicAlign64") },
	"slice.go":         nil,
	"subst.go":         func(f *ast.File) { fixTokenPos(f); fixTraceSel(f) },
	"termlist.go":      nil,
	"termlist_test.go": nil,
	"tuple.go":         nil,
	"typelists.go":     nil,
	"typeparam.go":     nil,
	"typeterm_test.go": nil,
	"typeterm.go":      nil,
	"under.go":         nil,
	"unify.go": func(f *ast.File) {
		fixSprintf(f)
		renameIdent(f, "EnableInterfaceInference", "_EnableInterfaceInference")
	},
	"universe.go":  fixGlobalTypVarDecl,
	"util_test.go": fixTokenPos,
	"validtype.go": nil,
}

// TODO(gri) We should be able to make these rewriters more configurable/composable.
//           For now this is a good starting point.

// renameIdent renames an identifier.
// Note: This doesn't change the use of the identifier in comments.
func renameIdent(f *ast.File, from, to string) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if n.Name == from {
				n.Name = to
			}
			return false
		}
		return true
	})
}

// renameImportPath renames an import path.
func renameImportPath(f *ast.File, from, to string) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			if n.Path.Kind == token.STRING && n.Path.Value == from {
				n.Path.Value = to
				return false
			}
		}
		return true
	})
}

// fixTokenPos changes imports of "cmd/compile/internal/syntax" to "go/token",
// uses of syntax.Pos to token.Pos, and calls to x.IsKnown() to x.IsValid().
func fixTokenPos(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			// rewrite import path "cmd/compile/internal/syntax" to "go/token"
			if n.Path.Kind == token.STRING && n.Path.Value == `"cmd/compile/internal/syntax"` {
				n.Path.Value = `"go/token"`
				return false
			}
		case *ast.SelectorExpr:
			// rewrite syntax.Pos to token.Pos
			if x, _ := n.X.(*ast.Ident); x != nil && x.Name == "syntax" && n.Sel.Name == "Pos" {
				x.Name = "token"
				return false
			}
		case *ast.CallExpr:
			// rewrite x.IsKnown() to x.IsValid()
			if fun, _ := n.Fun.(*ast.SelectorExpr); fun != nil && fun.Sel.Name == "IsKnown" && len(n.Args) == 0 {
				fun.Sel.Name = "IsValid"
				return false
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
			if n.Name.Name == "infer" || n.Name.Name == "infer1" || n.Name.Name == "infer2" {
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
					if ident, _ := n.Args[0].(*ast.Ident); ident != nil && ident.Name == "pos" {
						pos := n.Args[0].Pos()
						fun := &ast.SelectorExpr{X: newIdent(pos, "posn"), Sel: newIdent(pos, "Pos")}
						arg := &ast.CallExpr{Fun: fun, Lparen: pos, Args: nil, Ellipsis: token.NoPos, Rparen: pos}
						n.Args[0] = arg
						return false
					}
				case "errorf", "infer1", "infer2":
					// rewrite check.errorf(pos, ...) to check.errorf(posn, ...)
					// rewrite check.infer1(pos, ...) to check.infer1(posn, ...)
					// rewrite check.infer2(pos, ...) to check.infer2(posn, ...)
					if ident, _ := n.Args[0].(*ast.Ident); ident != nil && ident.Name == "pos" {
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

// fixErrErrorfCall updates calls of the form err.errorf(obj, ...) to err.errorf(obj.Pos(), ...).
func fixErrErrorfCall(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if selx, _ := n.Fun.(*ast.SelectorExpr); selx != nil {
				if ident, _ := selx.X.(*ast.Ident); ident != nil && ident.Name == "err" {
					switch selx.Sel.Name {
					case "errorf":
						// rewrite err.errorf(obj, ... ) to err.errorf(obj.Pos(), ... )
						if ident, _ := n.Args[0].(*ast.Ident); ident != nil && ident.Name == "obj" {
							pos := n.Args[0].Pos()
							fun := &ast.SelectorExpr{X: ident, Sel: newIdent(pos, "Pos")}
							arg := &ast.CallExpr{Fun: fun, Lparen: pos, Args: nil, Ellipsis: token.NoPos, Rparen: pos}
							n.Args[0] = arg
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
				if ident, _ := selx.X.(*ast.Ident); ident != nil && ident.Name == "check" {
					switch selx.Sel.Name {
					case "errorf":
						// rewrite check.errorf(pos, ... ) to check.errorf(atPos(pos), ... )
						if ident, _ := n.Args[0].(*ast.Ident); ident != nil && ident.Name == "pos" {
							pos := n.Args[0].Pos()
							fun := newIdent(pos, "atPos")
							arg := &ast.CallExpr{Fun: fun, Lparen: pos, Args: []ast.Expr{ident}, Ellipsis: token.NoPos, Rparen: pos}
							n.Args[0] = arg
							return false
						}
					}
				}
			}
		}
		return true
	})
}

// fixTraceSel renames uses of x.Trace to x.trace, where x for any x with a Trace field.
func fixTraceSel(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			// rewrite x.Trace to x._Trace (for Config.Trace)
			if n.Sel.Name == "Trace" {
				n.Sel.Name = "_Trace"
				return false
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
			if fun, _ := n.Fun.(*ast.Ident); fun != nil && fun.Name == "sprintf" && len(n.Args) >= 4 /* ... args */ {
				n.Args = insert(n.Args, 1, newIdent(n.Args[1].Pos(), "nil"))
				return false
			}
		}
		return true
	})
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
