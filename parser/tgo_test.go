package parser

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"

	"github.com/tgo-lang/lang/ast"
	"github.com/tgo-lang/lang/internal/defaulttgo"
	"github.com/tgo-lang/lang/scanner"
	"github.com/tgo-lang/lang/token"
)

func init() {
	defaulttgo.Enable()
}

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
				&ast.Text{
					StartPos: off,
					Text:     `"test"`,
				},
			},
		},
		{
			in: `"test \{sth}"`,
			out: []ast.Stmt{
				&ast.TemplateLiteral{
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
		{
			in: `"test \{sth} \{sth}"`,
			out: []ast.Stmt{
				&ast.TemplateLiteral{
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
		{
			in: `@attr`,
			out: []ast.Stmt{
				&ast.Attribute{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
				},
			},
		},
		{
			in: `@attr="test"`,
			out: []ast.Stmt{
				&ast.Attribute{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.Text{
						StartPos: off + 6,
						Text:     `"test"`,
					},
				},
			},
		},
		{
			in: `@attr="test \{sth}"`,
			out: []ast.Stmt{
				&ast.Attribute{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.TemplateLiteral{
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
				},
			},
		},
		{
			in: `@attr="test \{sth}t"`,
			out: []ast.Stmt{
				&ast.Attribute{
					StartPos: off,
					AttrName: &ast.Ident{
						NamePos: off + 1,
						Name:    "attr",
					},
					AssignPos: off + 5,
					Value: &ast.TemplateLiteral{
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
				},
			},
		},
		{
			in: `<div></div>`,
			out: []ast.Stmt{
				&ast.Element{
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
				&ast.Element{
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
						&ast.Text{
							StartPos: off + 5,
							Text:     `"test"`,
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
				&ast.Element{
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
						&ast.TemplateLiteral{
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
		f, err := ParseFile(fs, "test.go", inStr, SkipObjectResolution|ParseTgo)
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
			f, err := ParseFile(fs, filepath.Base(testFile), content, SkipObjectResolution|ParseComments|AllErrors|ParseTgo)
			if err != nil {
				if v.Name() != "element_blocks.tgo" {
					if v, ok := err.(scanner.ErrorList); ok {
						for _, err := range v {
							t.Logf("%v", err)
						}
					}
					t.Logf("Error while parsing file %v: %v", testFile, err)
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
		f, err := ParseFile(fs, name, src, SkipObjectResolution|ParseComments|ParseTgo)
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

		ast.Inspect(f, func(n ast.Node) bool {
			list := func(stmts []ast.Stmt) {
				for i := range stmts {
					stmt := &stmts[i]
					if v, ok := (*stmt).(*ast.Text); ok {
						*stmt = &ast.ExprStmt{
							X: &ast.BasicLit{
								ValuePos: v.StartPos,
								Kind:     token.STRING,
								Value:    v.Text,
							},
						}
					}
				}
			}
			switch n := n.(type) {
			case *ast.BlockStmt:
				list(n.List)
			case *ast.CaseClause:
				list(n.Body)
			case *ast.CommClause:
				list(n.Body)
			}
			return true
		})

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
		f, err := ParseFile(fs, name, src, SkipObjectResolution|ParseComments|ParseTgo)
		if err != nil {
			return
		}

		goParsable := true
		ast.Inspect(f, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.OpenTag, *ast.EndTag, *ast.Element,
				*ast.TemplateLiteral, *ast.Attribute:
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

// This test makes sure that we handle p.lineComment/p.leadComment properly in the [parser.next] method.
func TestTemplateLiteralsLineAndLeadComments(t *testing.T) {
	t.Run("lead comment", func(t *testing.T) {
		const src = "//comment\n" + `"\{1}"`
		fset := token.NewFileSet()
		file := fset.AddFile("test.go", -1, len(src))
		var p parser
		p.init(file, []byte(src), ParseComments|ParseTgo)
		if p.tok != token.STRING_TEMPLATE {
			t.Errorf("p.tok = %v; want = %v", p.tok, token.STRING_TEMPLATE)
		}
		if p.lineComment != nil {
			t.Errorf("p.leadComment = %v; want = <nil>", p.lineComment)
		}
		if p.leadComment == nil {
			t.Fatalf("p.leadComment = nil")
		}
		if len(p.leadComment.List) != 1 {
			t.Fatalf("len(p.leadComment.List) = %v; want = 1", len(p.leadComment.List))
		}
		if p.leadComment.List[0].Text != "//comment" {
			t.Fatalf(`p.leadComment.List[0].Text = %v; want = "//comment"`, p.leadComment.List[0].Text)
		}
	})
	t.Run("line comment", func(t *testing.T) {
		const src = "; //comment\n \"\\{1}\""
		fset := token.NewFileSet()
		file := fset.AddFile("test.go", -1, len(src))
		var p parser
		p.init(file, []byte(src), ParseComments|ParseTgo)
		p.next()
		if p.tok != token.STRING_TEMPLATE {
			t.Errorf("p.tok = %v; want = %v", p.tok, token.STRING_TEMPLATE)
		}
		if p.leadComment != nil {
			t.Errorf("p.leadComment= %v; want = <nil>", p.leadComment)
		}
		if p.lineComment == nil {
			t.Fatalf("p.lineComment = nil")
		}
		if len(p.lineComment.List) != 1 {
			t.Fatalf("len(p.lineComment.List) = %v; want = 1", len(p.lineComment.List))
		}
		if p.lineComment.List[0].Text != "//comment" {
			t.Fatalf(`p.lineComment.List[0].Text = %v; want = "//comment"`, p.lineComment.List[0].Text)
		}
	})
}

func TestTgoErrors(t *testing.T) {
	var cases = []string{
		`package p; func _() { "\{/* ERROR AFTER "expected operand, found '}'" */}" }`,
	}
	for _, src := range cases {
		checkErrors(t, src, src, DeclarationErrors|AllErrors|ParseTgo|SkipObjectResolution, true)
	}
}

func TestTgoParseExpr(t *testing.T) {
	defaulttgo.ExpectDisabled(t)

	fset := token.NewFileSet()
	expr, err := ParseExprFrom(fset, "test.tgo", `"test"`, SkipObjectResolution|ParseTgo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.BasicLit); !ok {
		t.Errorf("want = *ast.BasicLit; got = %v", expr)
	}

	expr, err = ParseExprFrom(fset, "test.tgo", `func() { "test" }`, SkipObjectResolution|ParseTgo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.FuncLit).Body.List[0].(*ast.Text); !ok {
		t.Errorf("want = *ast.Text ; got = %v", expr)
	}

	expr, err = ParseExprFrom(fset, "test.tgo", `func() { <div></div> }`, SkipObjectResolution|ParseTgo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.FuncLit).Body.List[0].(*ast.Element); !ok {
		t.Errorf("want = *ast.Element; got = %v", expr)
	}

	expr, err = ParseExprFrom(fset, "test.tgo", `func() { <br> }`, SkipObjectResolution|ParseTgo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.FuncLit).Body.List[0].(*ast.OpenTag); !ok {
		t.Errorf("want = *ast.OpenTag; got = %v", expr)
	}

	expr, err = ParseExprFrom(fset, "test.go", `func() { "test" }`, SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.FuncLit).Body.List[0].(*ast.ExprStmt); !ok {
		t.Errorf("want = *ast.ExprStmt; got = %v", expr)
	}
	expr, err = ParseExpr(`func() { "test" }`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*ast.FuncLit).Body.List[0].(*ast.ExprStmt); !ok {
		t.Errorf("want = *ast.ExprStmt; got = %v", expr)
	}

	_, err = ParseExprFrom(fset, "test.go", `func() { <br> }`, SkipObjectResolution)
	if err == nil {
		t.Fatal("ParseExprFrom() = <nil>")
	}
	_, err = ParseExpr(`func() { <br> }`)
	if err == nil {
		t.Fatal("ParseExpr() = <nil>")
	}
}

func TestTgoParseDir(t *testing.T) {
	defaulttgo.ExpectDisabled(t)

	type file struct {
		name string
		src  string
	}

	prepareFiles := func(t *testing.T, files []file) string {
		dir := t.TempDir()
		for _, file := range files {
			path := filepath.Join(dir, file.name)
			if err := os.WriteFile(path, []byte(file.src), 06660); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	dir := prepareFiles(t, []file{
		{name: "file.go", src: `package test; func test() { "test" }`},
		{name: "file.tgo", src: `package test; func test() { "test" }`},
		{name: "file2.tgo", src: `package test; func test() { "test" }`},
		{name: "file1.go", src: `package test; func test() { "test" }`},
	})

	t.Run("non tgo mode", func(t *testing.T) {
		fset := token.NewFileSet()
		pkgs, err := ParseDir(fset, dir, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		if len(pkgs) != 1 {
			t.Fatalf("len(pkgs) = %v; want = 1", len(pkgs))
		}
		var files map[string]*ast.File
		for _, pkg := range pkgs {
			files = pkg.Files
		}

		wantFiles := []string{
			filepath.Join(dir, "file.go"),
			filepath.Join(dir, "file1.go"),
		}
		gotFiles := slices.Sorted(maps.Keys(files))
		if !slices.Equal(wantFiles, gotFiles) {
			t.Fatalf("got = %v; want = %v", gotFiles, wantFiles)
		}

		// [ParseTgo] bit is cleared before being passed to [ParseFile],
		// so we should not get any *ast.Text nodes.
		for file, v := range files {
			ast.Inspect(v, func(n ast.Node) bool {
				switch n.(type) {
				case *ast.Text:
					t.Errorf("*ast.Text in file: %v", file)
				}
				return true
			})
		}
	})

	t.Run("tgo mode", func(t *testing.T) {
		fset := token.NewFileSet()
		pkgs, err := ParseDir(fset, dir, nil, ParseTgo)
		if err != nil {
			t.Fatal(err)
		}

		if len(pkgs) != 1 {
			t.Fatalf("len(pkgs) = %v; want = 1", len(pkgs))
		}
		var files map[string]*ast.File
		for _, pkg := range pkgs {
			files = pkg.Files
		}

		wantFiles := []string{
			filepath.Join(dir, "file.tgo"),
			filepath.Join(dir, "file1.go"),
			filepath.Join(dir, "file2.tgo"),
		}
		gotFiles := slices.Sorted(maps.Keys(files))
		if !slices.Equal(wantFiles, gotFiles) {
			t.Fatalf("got = %v; want = %v", gotFiles, wantFiles)
		}

		// [ParseTgo] bit is cleared before being passed to [ParseFile],
		// so we should get *ast.Text only in ".tgo" files.
		for file, v := range files {
			switch filepath.Ext(file) {
			case ".tgo":
				ast.Inspect(v, func(n ast.Node) bool {
					switch n.(type) {
					case *ast.ExprStmt:
						t.Errorf("*ast.ExprStmt in file: %v", file)
					}
					return true
				})
			case ".go":
				ast.Inspect(v, func(n ast.Node) bool {
					switch n.(type) {
					case *ast.Text:
						t.Errorf("*ast.Text in file: %v", file)
					}
					return true
				})
			default:
				t.Fatalf("unexpected file extension in file: %q", file)
			}
		}
	})

	t.Run("non-tgo mode invalid tgo file", func(t *testing.T) {
		dir := prepareFiles(t, []file{
			{name: "file.go", src: `package test; func test() { "test" }`},
			{name: "file.tgo", src: `invalid file`},
		})
		_, err := ParseDir(token.NewFileSet(), dir, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("tgo mode invalid go file", func(t *testing.T) {
		dir := prepareFiles(t, []file{
			{name: "file.go", src: `invalid file`},
			{name: "file.tgo", src: `package test; func test() { "test" }`},
		})
		_, err := ParseDir(token.NewFileSet(), dir, nil, ParseTgo)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("tgo mode invalid go file 2", func(t *testing.T) {
		dir := prepareFiles(t, []file{
			{name: "file2.go", src: `invalid file`},
			{name: "file.tgo", src: `package test; func test() { "test" }`},
		})
		_, err := ParseDir(token.NewFileSet(), dir, nil, ParseTgo)
		if err == nil {
			t.Fatal("unexpected success")
		}
	})

	t.Run("non tgo mode filter", func(t *testing.T) {
		fset := token.NewFileSet()
		_, err := ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			if filepath.Ext(fi.Name()) != ".go" {
				t.Errorf("unexpected file extension reached the filter: %q", fi.Name())
			}
			return true
		}, 0)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("tgo mode filter", func(t *testing.T) {
		fset := token.NewFileSet()
		pkgs, err := ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			return filepath.Ext(fi.Name()) == ".go"
		}, ParseTgo)
		if err != nil {
			t.Fatal(err)
		}

		if len(pkgs) != 1 {
			t.Fatalf("len(pkgs) = %v; want = 1", len(pkgs))
		}
		var files map[string]*ast.File
		for _, pkg := range pkgs {
			files = pkg.Files
		}

		wantFiles := []string{filepath.Join(dir, "file1.go")}
		gotFiles := slices.Sorted(maps.Keys(files))
		if !slices.Equal(wantFiles, gotFiles) {
			t.Fatalf("got = %v; want = %v", gotFiles, wantFiles)
		}
	})

	t.Run("tgo mode filter2", func(t *testing.T) {
		fset := token.NewFileSet()
		pkgs, err := ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			return filepath.Ext(fi.Name()) == ".tgo"
		}, ParseTgo)
		if err != nil {
			t.Fatal(err)
		}

		if len(pkgs) != 1 {
			t.Fatalf("len(pkgs) = %v; want = 1", len(pkgs))
		}
		var files map[string]*ast.File
		for _, pkg := range pkgs {
			files = pkg.Files
		}

		wantFiles := []string{
			filepath.Join(dir, "file.tgo"),
			filepath.Join(dir, "file2.tgo"),
		}
		gotFiles := slices.Sorted(maps.Keys(files))
		if !slices.Equal(wantFiles, gotFiles) {
			t.Fatalf("got = %v; want = %v", gotFiles, wantFiles)
		}
	})
}

