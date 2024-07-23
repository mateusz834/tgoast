package parser

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/scanner"
	"github.com/mateusz834/tgoast/token"
)

func TestTgoBasicSyntax(t *testing.T) {
	const prefix = "package main\nfunc test() {"
	off := token.Pos(len(prefix)) + 1

	cases := []struct {
		in  string
		out []ast.Stmt
	}{
		{
			in: `<div>`,
			out: []ast.Stmt{
				&ast.OpenTagStmt{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 1,
						Name:    "div",
					},
					Body:     nil,
					ClosePos: off + 4,
				},
			},
		},
		{
			in: `</div>`,
			out: []ast.Stmt{
				&ast.EndTagStmt{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 2,
						Name:    "div",
					},
					ClosePos: off + 5,
				},
			},
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
						Parts: []ast.Expr{
							&ast.Ident{
								NamePos: off + 8,
								Name:    "sth",
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
						Parts: []ast.Expr{
							&ast.Ident{
								NamePos: off + 8,
								Name:    "sth",
							},
							&ast.Ident{
								NamePos: off + 15,
								Name:    "sth",
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
						Parts: []ast.Expr{
							&ast.Ident{
								NamePos: off + 14,
								Name:    "sth",
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
						Parts: []ast.Expr{
							&ast.Ident{
								NamePos: off + 14,
								Name:    "sth",
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
				&ast.OpenTagStmt{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 1,
						Name:    "div",
					},
					Body:     nil,
					ClosePos: off + 4,
				},
				&ast.EndTagStmt{
					OpenPos: off + 5,
					Name: &ast.Ident{
						NamePos: off + 7,
						Name:    "div",
					},
					ClosePos: off + 10,
				},
			},
		},
		{
			in: `<div>"test"</div>`,
			out: []ast.Stmt{
				&ast.OpenTagStmt{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 1,
						Name:    "div",
					},
					Body:     nil,
					ClosePos: off + 4,
				},
				&ast.ExprStmt{
					X: &ast.BasicLit{
						ValuePos: off + 5,
						Kind:     token.STRING,
						Value:    `"test"`,
					},
				},
				&ast.EndTagStmt{
					OpenPos: off + 11,
					Name: &ast.Ident{
						NamePos: off + 13,
						Name:    "div",
					},
					ClosePos: off + 16,
				},
			},
		},
		{
			in: `<div>"test \{sth}"</div>`,
			out: []ast.Stmt{
				&ast.OpenTagStmt{
					OpenPos: off,
					Name: &ast.Ident{
						NamePos: off + 1,
						Name:    "div",
					},
					Body:     nil,
					ClosePos: off + 4,
				},
				&ast.ExprStmt{
					X: &ast.TemplateLiteralExpr{
						OpenPos: off + 5,
						Strings: []string{
							`"test `,
							`"`,
						},
						Parts: []ast.Expr{
							&ast.Ident{
								NamePos: off + 13,
								Name:    "sth",
							},
						},
						ClosePos: off + 17,
					},
				},
				&ast.EndTagStmt{
					OpenPos: off + 18,
					Name: &ast.Ident{
						NamePos: off + 20,
						Name:    "div",
					},
					ClosePos: off + 23,
				},
			},
		},
	}

	for _, tt := range cases {
		inStr := prefix + tt.in + "}"

		fs := token.NewFileSet()
		f, err := ParseFile(fs, "test.go", inStr, SkipObjectResolution)
		if err != nil {
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
			var out strings.Builder
			ast.Fprint(&out, fs, f, nil)
			t.Logf("\n%v", out.String())
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
			f, err := ParseFile(fs, filepath.Base(testFile), content, SkipObjectResolution|ParseComments)
			if err != nil {
				if v, ok := err.(scanner.ErrorList); ok {
					for _, err := range v {
						t.Errorf("%v", err)
					}
				}
				t.Errorf("Error while parsing file %v: %v", testFile, err)
				continue
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
				d, err := diff(t.TempDir(), string(expect), got)
				if err == nil {
					t.Logf("\n%v", d)
				}
			}
		}
	}

}

func diff(tmpDir string, got, expect string) (string, error) {
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
