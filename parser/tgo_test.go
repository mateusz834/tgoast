package parser

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/scanner"
	"github.com/mateusz834/tgoast/token"
)

func TestTgoBasicSyntax(t *testing.T) {
	const prefix = "package main\nfunc test() {"
	off := token.Pos(len(prefix)) + 1

	cases := []struct {
		in    string
		out   []ast.Stmt
		errOk bool
	}{
		{
			in: `<div>`,
			out: []ast.Stmt{
				&ast.OpenTag{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 1,
						Name:    "div",
					},
					Body:     nil,
					ClosePos: off + 4,
				},
			},
			errOk: true,
		},
		{
			in: `</div>`,
			out: []ast.Stmt{
				&ast.EndTag{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 2,
						Name:    "div",
					},
					ClosePos: off + 5,
				},
			},
			errOk: true,
		},
		{
			in: `"test"`,
			out: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.BasicLit{
						ValuePos: off,
						Kind:     token.STRING,
						Value:    `"test"`,
					},
				},
			},
		},
		{
			in: `"test \{sth}"`,
			out: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.TemplateLiteralExpr{
						OpenPos: off,
						Strings: []string{
							`"test `,
							`"`,
						},
						Parts: []*ast.TemplateLiteralPart{
							{
								LBrace: off + 7,
								X: &ast.Ident{
									NamePos: off + 8,
									Name:    "sth",
								},
								RBrace: off + 11,
							},
						},
						ClosePos: off + 12,
					},
				},
			},
		},
		{
			in: `"test \{sth} \{sth}"`,
			out: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.TemplateLiteralExpr{
						OpenPos: off,
						Strings: []string{
							`"test `,
							` `,
							`"`,
						},
						Parts: []*ast.TemplateLiteralPart{
							{
								LBrace: off + 7,
								X: &ast.Ident{
									NamePos: off + 8,
									Name:    "sth",
								},
								RBrace: off + 11,
							},
							{
								LBrace: off + 14,
								X: &ast.Ident{
									NamePos: off + 15,
									Name:    "sth",
								},
								RBrace: off + 18,
							},
						},
						ClosePos: off + 19,
					},
				},
			},
		},
		{
			in: `@attr`,
			out: []ast.Stmt{
				&ast.AttributeStmt{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					EndPos: off + 4,
				},
			},
		},
		{
			in: `@attr="test"`,
			out: []ast.Stmt{
				&ast.AttributeStmt{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.BasicLit{
						ValuePos: off + 6,
						Kind:     token.STRING,
						Value:    `"test"`,
					},
					EndPos: off + 11,
				},
			},
		},
		{
			in: `@attr="test \{sth}"`,
			out: []ast.Stmt{
				&ast.AttributeStmt{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.TemplateLiteralExpr{
						OpenPos: off + 6,
						Strings: []string{
							`"test `,
							`"`,
						},
						Parts: []*ast.TemplateLiteralPart{
							{
								LBrace: off + 13,
								X: &ast.Ident{
									NamePos: off + 14,
									Name:    "sth",
								},
								RBrace: off + 17,
							},
						},
						ClosePos: off + 18,
					},
					EndPos: off + 18,
				},
			},
		},
		{
			in: `@attr="test \{sth}t"`,
			out: []ast.Stmt{
				&ast.AttributeStmt{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.TemplateLiteralExpr{
						OpenPos: off + 6,
						Strings: []string{
							`"test `,
							`t"`,
						},
						Parts: []*ast.TemplateLiteralPart{
							{
								LBrace: off + 13,
								X: &ast.Ident{
									NamePos: off + 14,
									Name:    "sth",
								},
								RBrace: off + 17,
							},
						},
						ClosePos: off + 19,
					},
					EndPos: off + 19,
				},
			},
		},
		{
			in: `<div></div>`,
			out: []ast.Stmt{
				&ast.ElementBlockStmt{
					OpenTag: &ast.OpenTag{
						OpenPos: off,
						Name: &ast.Ident{
							NamePos: off + 1,
							Name:    "div",
						},
						Body:     nil,
						ClosePos: off + 4,
					},
					EndTag: &ast.EndTag{
						OpenPos: off + 5,
						Name: &ast.Ident{
							NamePos: off + 7,
							Name:    "div",
						},
						ClosePos: off + 10,
					},
				},
			},
		},
		{
			in: `<div>"test"</div>`,
			out: []ast.Stmt{
				&ast.ElementBlockStmt{
					OpenTag: &ast.OpenTag{
						OpenPos: off,
						Name: &ast.Ident{
							NamePos: off + 1,
							Name:    "div",
						},
						Body:     nil,
						ClosePos: off + 4,
					},
					Body: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.BasicLit{
								ValuePos: off + 5,
								Kind:     token.STRING,
								Value:    `"test"`,
							},
						},
					},
					EndTag: &ast.EndTag{
						OpenPos: off + 11,
						Name: &ast.Ident{
							NamePos: off + 13,
							Name:    "div",
						},
						ClosePos: off + 16,
					},
				},
			},
		},
		{
			in: `<div>"test \{sth}"</div>`,
			out: []ast.Stmt{
				&ast.ElementBlockStmt{
					OpenTag: &ast.OpenTag{
						OpenPos: off,
						Name: &ast.Ident{
							NamePos: off + 1,
							Name:    "div",
						},
						Body:     nil,
						ClosePos: off + 4,
					},
					Body: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.TemplateLiteralExpr{
								OpenPos: off + 5,
								Strings: []string{
									`"test `,
									`"`,
								},
								Parts: []*ast.TemplateLiteralPart{
									{
										LBrace: off + 12,
										X: &ast.Ident{
											NamePos: off + 13,
											Name:    "sth",
										},
										RBrace: off + 16,
									},
								},
								ClosePos: off + 17,
							},
						},
					},
					EndTag: &ast.EndTag{
						OpenPos: off + 18,
						Name: &ast.Ident{
							NamePos: off + 20,
							Name:    "div",
						},
						ClosePos: off + 23,
					},
				},
			},
		},
	}

	for _, tt := range cases {
		inStr := prefix + tt.in + "}"

		fs := token.NewFileSet()
		f, err := ParseFile(fs, "test.go", inStr, SkipObjectResolution)
		if err != nil && !tt.errOk {
			t.Errorf("%v: unexpected error: %v", inStr, err)
		}

		if len(f.Decls) == 0 {
			t.Errorf("missing func decl")
			continue
		}
		fd, ok := f.Decls[0].(*ast.FuncDecl)
		if !ok {
			t.Errorf("f.Decls[0] is not *ast.FuncDecl")
			continue
		}

		expectList := fd.Body.List
		if !reflect.DeepEqual(expectList, tt.out) {
			t.Errorf("unexpected AST for:\n%v", inStr)
			var out, want strings.Builder
			ast.Fprint(&out, fs, f.Decls[0].(*ast.FuncDecl).Body.List, nil)
			ast.Fprint(&want, fs, tt.out, nil)
			t.Logf("\n%v", out.String())
			t.Logf("want:\n%v", want.String())
			//diff, _ := gitDiff(t.TempDir(), out.String(), want.String())
			//t.Logf("diff:\n%v", diff)k
		}
	}
}

func TestTgoSyntax(t *testing.T) {
	const testdata = "./testdata/tgo"
	files, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range files {
		ext := filepath.Ext(v.Name())
		if ext == ".tgo" {
			testFile := filepath.Join(testdata, v.Name())
			expectFileName := filepath.Join(testdata, v.Name()[:len(v.Name())-len(".tgo")]+".ast")

			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			fs := token.NewFileSet()
			f, err := ParseFile(fs, filepath.Base(testFile), content, SkipObjectResolution|ParseComments|AllErrors)
			if err != nil {
				if v, ok := err.(scanner.ErrorList); ok {
					for _, err := range v {
						t.Logf("%v", err)
					}
				}
				t.Logf("Error while parsing file %v: %v", testFile, err)
				if v.Name() != "element_blocks.tgo" {
					t.Fail()
					continue
				}
			}

			var b strings.Builder
			ast.Fprint(&b, fs, f, ast.NotNilFilter)

			expect, err := os.ReadFile(expectFileName)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if err := os.WriteFile(expectFileName, []byte(b.String()), 06660); err != nil {
						t.Fatal(err)
					}
					continue
				}
				t.Fatal(err)
			}

			got := b.String()
			if string(expect) != got {
				t.Errorf("unexpected in %v", testFile)
				d, err := gitDiff(t.TempDir(), string(expect), got)
				if err == nil {
					t.Logf("\n%v", d)
				}
			}
		}
	}

}

func fuzzAddDir(f *testing.F, testdata string) {
	files, err := os.ReadDir(testdata)
	if err != nil {
		f.Fatal(err)
	}
	for _, v := range files {
		if v.IsDir() {
			continue
		}

		testFile := filepath.Join(testdata, v.Name())
		content, err := os.ReadFile(testFile)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(testFile, string(content))
	}
}

func FuzzGoParsableByTgo(f *testing.F) {
	fuzzAddDir(f, "../printer")
	fuzzAddDir(f, "../printer/testdata")
	fuzzAddDir(f, "../parser")
	fuzzAddDir(f, "../parser/testdata")
	fuzzAddDir(f, "../ast")
	f.Fuzz(func(t *testing.T, name, src string) {
		gfs := gotoken.NewFileSet()
		gf, err := goparser.ParseFile(gfs, name, src, goparser.SkipObjectResolution|goparser.ParseComments)
		if err != nil {
			return
		}

		fs := token.NewFileSet()
		f, err := ParseFile(fs, name, src, SkipObjectResolution|ParseComments)
		if err != nil {
			t.Fatalf("ParseFile() = %v; want = <nil>", err)
		}

		var (
			goAst  strings.Builder
			tgoAst strings.Builder
		)

		if err := goast.Fprint(&goAst, gfs, gf, nil); err != nil {
			t.Fatalf("goast.Fprint() = %v; want = <nil>", err)
		}

		if err := ast.Fprint(&tgoAst, fs, f, nil); err != nil {
			t.Fatalf("ast.Fprint() = %v; want = <nil>", err)
		}

		if goAst.String() != tgoAst.String() {
			diff, err := gitDiff(t.TempDir(), goAst.String(), tgoAst.String())
			if err != nil {
				t.Fatalf("difference found")
			}
			t.Fatalf(
				"difference found, apply following changes to make this test pass:\n%v",
				diff,
			)
		}
	})
}

func FuzzTgoNotParsableByGo(f *testing.F) {
	fuzzAddDir(f, "../printer/testdata/tgo")
	fuzzAddDir(f, "../parser/testdata/tgo")
	fuzzAddDir(f, "../printer")
	fuzzAddDir(f, "../printer/testdata")
	fuzzAddDir(f, "../parser")
	fuzzAddDir(f, "../parser/testdata")
	fuzzAddDir(f, "../ast")
	f.Fuzz(func(t *testing.T, name, src string) {
		fs := token.NewFileSet()
		f, err := ParseFile(fs, name, src, SkipObjectResolution|ParseComments)
		if err != nil {
			return
		}

		goParsable := true
		ast.Inspect(f, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.OpenTag, *ast.EndTag, *ast.ElementBlockStmt,
				*ast.TemplateLiteralExpr, *ast.AttributeStmt:
				goParsable = false
			}
			return true
		})

		if !goParsable {
			gfs := gotoken.NewFileSet()
			_, err = goparser.ParseFile(gfs, name, src, goparser.SkipObjectResolution|goparser.ParseComments)
			if err == nil {
				t.Fatalf("ParseFile() = <nil>; want = (not <nil>)")
			}
		}
	})
}

func gitDiff(tmpDir string, got, expect string) (string, error) {
	gotPath := filepath.Join(tmpDir, "got")
	gotFile, err := os.Create(gotPath)
	if err != nil {
		return "", err
	}
	defer gotFile.Close()
	if _, err := gotFile.WriteString(got); err != nil {
		return "", err
	}

	expectPath := filepath.Join(tmpDir, "expect")
	expectFile, err := os.Create(expectPath)
	if err != nil {
		return "", err
	}
	defer expectFile.Close()
	if _, err := expectFile.WriteString(expect); err != nil {
		return "", err
	}

	var out strings.Builder
	cmd := exec.Command("git", "diff", "-U 100000", "--no-index", "--color=always", "--ws-error-highlight=all", gotPath, expectPath)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil && cmd.ProcessState.ExitCode() != 1 {
		return "", err
	}
	return out.String(), nil
}
