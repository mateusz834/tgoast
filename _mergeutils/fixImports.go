//go:build ignore

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"
	gofumpt "mvdan.cc/gofumpt/format"
)

const module = "github.com/tgo-lang/lang"

var importsMapper = map[string]string{
	"go/ast":      module + "/ast",
	"go/constant": module + "/constant",
	"go/doc":      module + "/doc",
	"go/format":   module + "/format",
	"go/importer": module + "/importer",
	"go/parser":   module + "/parser",
	"go/printer":  module + "/printer",
	"go/scanner":  module + "/scanner",
	"go/token":    module + "/token",
	"go/types":    module + "/types",

	"go/internal/gccgoimporter": module + "/internal/go/gccgoimporter",
	"go/internal/gcimporter":    module + "/internal/go/gcimporter",
	"go/internal/srcimporter":   module + "/internal/go/srcimporter",
	"go/internal/typeparams":    module + "/internal/go/typeparams",

	"internal/bisect":   module + "/internal/bisect",
	"internal/buildcfg": module + "/internal/buildcfg",
	"internal/cfg":      module + "/internal/cfg",
	"internal/diff":     module + "/internal/diff",
	"internal/goarch":   module + "/internal/goarch",
	"internal/godebug":  module + "/internal/godebug",
	//"internal/godebugs":     module + "/internal/godebugs",
	"internal/goexperiment": module + "/internal/goexperiment",
	"internal/goversion":    module + "/internal/goversion",
	"internal/lazyregexp":   module + "/internal/lazyregexp",
	"internal/pkgbits":      module + "/internal/pkgbits",
	"internal/platform":     module + "/internal/platform",
	//"internal/race":         module + "/internal/race",
	"internal/saferio":      module + "/internal/saferio",
	"internal/testenv":      module + "/internal/testenv",
	"internal/txtar":        module + "/internal/txtar",
	"internal/types/errors": module + "/internal/types/errors",
	"internal/xcoff":        module + "/internal/xcoff",
	"internal/exportdata":   module + "/internal/exportdata",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func updatedImport(file, path string) (string, bool) {
	if (strings.Contains(file, "token/position.go") && path == "go/token") || strings.Contains(file, "parser/tgo_test.go") {
		return path, false
	}
	if v, ok := importsMapper[path]; ok {
		return v, true
	}
	return path, false
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("missing branch name")
	}
	branch := os.Args[1]

	slog.Info("auto merge conflict of imports")
	out, err := cmd("git", "ls-files", "--unmerged", "--format=%(path)")
	if err != nil {
		return err
	}
	for _, file := range slices.Compact(strings.Split(out, "\n")) {
		err := updateImports(file, branch, func(fset *token.FileSet, f *ast.File) {
		outer:
			for _, imp := range f.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					panic(err) // AST is valid
				}
				if v, ok := updatedImport(file, path); ok {
					v = strconv.Quote(v)
					for _, imp2 := range f.Imports {
						if imp2.Path != nil && v == imp2.Path.Value && imp != imp2 {
							var n1, n2 string
							if imp.Name != nil {
								n1 = imp.Name.Name
							}
							if imp2.Name != nil {
								n2 = imp2.Name.Name
							}
							if n1 == n2 {
								fset.File(f.FileStart).MergeLine(fset.Position(imp.Pos()).Line)
								imp.Path = nil // already imported
								continue outer
							}
						}
					}
					imp.Path.Value = v
				}
			}
			for _, v := range f.Decls {
				if v, ok := v.(*ast.GenDecl); ok && v.Tok == token.IMPORT {
				outer2:
					for {
						for i, spec := range v.Specs {
							spec := spec.(*ast.ImportSpec)
							if spec.Path == nil {
								v.Specs = slices.Delete(v.Specs, i, i+1)
								continue outer2
							}
						}
						break
					}
				}
			}
		})
		if err != nil {
			return err
		}
	}

	slog.Info("fix std imports")
	unknownImports := make(map[string][]string)
	err = filepath.WalkDir(".", func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return updateImports(file, branch, func(fset *token.FileSet, f *ast.File) {
			for _, v := range f.Decls {
				if v, ok := v.(*ast.GenDecl); ok && v.Tok == token.IMPORT {
					for _, spec := range v.Specs {
						spec := spec.(*ast.ImportSpec)
						path, err := strconv.Unquote(spec.Path.Value)
						if err != nil {
							panic(err) // AST is valid
						}
						if v, ok := updatedImport(file, path); ok {
							spec.Path.Value = strconv.Quote(v)
						} else if strings.Contains(path, "internal/") && !strings.HasPrefix(path, module) {
							f := fset.Position(f.FileStart).Filename
							unknownImports[f] = append(unknownImports[f], path)
						}
					}
				}
			}
		})
	})
	if err != nil {
		return err
	}

	for file, imports := range unknownImports {
		for _, imp := range imports {
			slog.Warn("unresolved internal import", "file", file, "import", imp)
		}
	}

	return nil
}

func updateImports(path, branch string, modify func(fset *token.FileSet, f *ast.File)) error {
	if strings.HasPrefix(path, ".") || path == "" {
		return nil
	}

	slog := slog.With("path", path)
	slog.Info("processing")

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if strings.Contains(path, "/testdata/") || info.IsDir() || filepath.Ext(path) != ".go" {
		slog.Info("skipping")
		return nil
	}

	fset := token.NewFileSet()
	fileContents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, _ := parser.ParseFile(fset, path, fileContents, parser.SkipObjectResolution|parser.ParseComments)
	for _, v := range f.Comments {
		for _, v := range v.List {
			c, err := constraint.Parse(v.Text)
			if err == nil {
				if c, ok := c.(*constraint.TagExpr); ok && c.Tag == "ignore" {
					slog.Info("//go:build ignore, skipping")
					return nil
				}
			}
		}
	}

	var last ast.Decl
	for _, v := range f.Decls {
		if v, ok := v.(*ast.GenDecl); !ok || v.Tok != token.IMPORT {
			break
		}
		last = v
	}

	if last == nil {
		slog.Info("no imports, skipping")
		return nil
	}

	f, err = parser.ParseFile(fset, path, removeMergeConflicts(string(fileContents[:last.End()]), branch), parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		slog.Info("invalid imports after removal of merge conflicts signs, skipping")
		return nil
	}

	modify(fset, f)

	f.Imports = nil
	var out strings.Builder
	if err := format.Node(&out, fset, f); err != nil {
		return err
	}
	f, err = parser.ParseFile(fset, path, out.String(), parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return err
	}
	gofumpt.File(fset, f, gofumpt.Options{
		LangVersion: "go1.24.0",
		ModulePath:  module,
	})
	out.Reset()
	if err := format.Node(&out, fset, f); err != nil {
		return err
	}
	outFmtImports, err := imports.Process(path, []byte(out.String()), &imports.Options{
		FormatOnly: true,
		Comments:   true,
	})
	if err != nil {
		return err
	}

	newFileContents := append(outFmtImports, fileContents[last.End():]...)

	if !bytes.Equal(fileContents, newFileContents) {
		if err := os.WriteFile(path, newFileContents, 0); err != nil {
			return err
		}
		slog.Info("updated imports")
	} else {
		slog.Info("imports already fine")
	}

	if removeMergeConflicts(string(newFileContents), branch) == string(newFileContents) {
		// No other conflicts.
		if err := cmdNoCapture("git", "add", path); err != nil {
			return err
		}
	}

	return nil
}

func removeMergeConflicts(s, branch string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				s,
				"<<<<<<< HEAD\n", "",
			),
			"=======\n", "",
		),
		">>>>>>> "+branch+"\n", "",
	)
}

func cmd(name string, args ...string) (string, error) {
	slog.Info("exec", "cmd", name+" "+strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func cmdNoCapture(name string, args ...string) error {
	slog.Info("exec", "cmd", name+" "+strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
