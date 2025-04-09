package types_test

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/tgo-lang/lang/ast"
	"github.com/tgo-lang/lang/internal/types/errors"
	"github.com/tgo-lang/lang/parser"
	"github.com/tgo-lang/lang/token"
	. "github.com/tgo-lang/lang/types"
)

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
	f, err := parser.ParseFile(fset, "pkg.go", src, parser.SkipObjectResolution|parser.ParseTgo)
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

	cfg := Config{Importer: defaultImporter(fset)}
	pkg, err := cfg.Check("pkg", fset, []*ast.File{f}, &infos)
	if err != nil {
		t.Fatal(err)
	}

	fun := f.Decls[1].(*ast.FuncDecl)
	article := fun.Body.List[0].(*ast.Element)
	div := article.Body[2].(*ast.Element)

	articleOpenTagAttrTemplateLit := article.OpenTag.Body[1].(*ast.Attribute).Value.(*ast.TemplateLiteral)
	articleTemplateLit := article.Body[1].(*ast.TemplateLiteral)
	divTemplateLit1 := div.Body[1].(*ast.TemplateLiteral)
	divTemplateLit2 := div.Body[2].(*ast.TemplateLiteral)
	divTemplateLit3 := div.Body[4].(*ast.TemplateLiteral)

	t.Run("types", func(t *testing.T) {
		wantTypes := map[ast.Expr]Type{
			articleOpenTagAttrTemplateLit.Parts[0].X: Typ[Int],
			articleOpenTagAttrTemplateLit.Parts[1].X: Typ[String],
			articleTemplateLit.Parts[0].X:            Typ[String],
			divTemplateLit1.Parts[0].X:               Typ[Int],
			divTemplateLit2.Parts[0].X:               Typ[String],
			divTemplateLit2.Parts[1].X:               Typ[Int],
			divTemplateLit3.Parts[0].X:               Typ[Int],
			divTemplateLit3.Parts[1].X:               Typ[Int],
			divTemplateLit3.Parts[2].X:               Typ[String],
			divTemplateLit3.Parts[3].X:               Typ[String],
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

func TestTgoScopes(t *testing.T) {
	const src = `package test

import "github.com/mateusz834/tgo"

func test(tgo.Ctx) error {
	<div>
		"test"
	</div>
	return nil
}

func test2(tgo.Ctx) error {
	<br>
	return nil
}

func test3(tgo.Ctx) error {
	<div>
		<br>
		<span>
			"test"
			a2 := 2
			_ = a2
		</span>
		a1 := 1
		_ = a1
	</div>
	return nil
}
`

	cases := []struct {
		name  string
		src   string
		valid bool
	}{
		{"valid src", src, true},
		{"non tgo funcs with tgo imported", strings.ReplaceAll(src, "(tgo.Ctx)", "()"), false},
		{
			"non tgo funcs tgo not imported",
			strings.ReplaceAll(strings.ReplaceAll(src, "(tgo.Ctx)", "()"), `import "github.com/mateusz834/tgo"`, `import "invalid"`),
			false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.tgo", tt.src, parser.SkipObjectResolution|parser.ParseTgo)
			if err != nil {
				t.Fatal(err)
			}

			cfg := Config{
				Importer: defaultImporter(fset),
				Error: func(err error) {
					if tt.valid {
						t.Fatal(err)
					}
				},
			}
			infos := Info{Scopes: make(map[ast.Node]*Scope)}
			pkg, err := cfg.Check("test", fset, []*ast.File{f}, &infos)
			if err != nil && tt.valid {
				t.Fatal(err)
			}

			elements := []*ast.Element{
				f.Decls[1].(*ast.FuncDecl).Body.List[0].(*ast.Element),
				f.Decls[3].(*ast.FuncDecl).Body.List[0].(*ast.Element),
				f.Decls[3].(*ast.FuncDecl).Body.List[0].(*ast.Element).Body[1].(*ast.Element),
			}

			fScope := pkg.Scope().Child(0)
			if c := fScope.Child(0).Child(1); c != infos.Scopes[elements[0]] {
				t.Errorf("fScope.Child(0).Child(1) = %v; want = %v", c, infos.Scopes[elements[0]])
			}
			if c := fScope.Child(2).Child(1); c != infos.Scopes[elements[1]] {
				t.Errorf("fScope.Child(2).Child(1) = %v; want = %v", c, infos.Scopes[elements[1]])
			}
			if c := fScope.Child(2).Child(1).Child(2); c != infos.Scopes[elements[2]] {
				t.Errorf("fScope.Child(2).Child(1).Child(2) = %v; want = %v", c, infos.Scopes[elements[2]])
			}

			if c := fScope.Child(0).Child(0); c != infos.Scopes[elements[0].OpenTag] {
				t.Errorf("fScope.Child(0).Child(0) = %v; want = %v", c, infos.Scopes[elements[0].OpenTag])
			}
			if c := fScope.Child(2).Child(0); c != infos.Scopes[elements[1].OpenTag] {
				t.Errorf("fScope.Child(2).Child(0) = %v; want = %v", c, infos.Scopes[elements[1].OpenTag])
			}
			if c := fScope.Child(2).Child(1).Child(1); c != infos.Scopes[elements[2].OpenTag] {
				t.Errorf("fScope.Child(2).Child(1).Child(1) = %v; want = %v", c, infos.Scopes[elements[2].OpenTag])
			}

			tag := f.Decls[2].(*ast.FuncDecl).Body.List[0].(*ast.OpenTag)
			if infos.Scopes[tag] == nil {
				t.Error("infos.Scopes[tag] = nil")
			}
			if c := fScope.Child(1).Child(0); c != infos.Scopes[tag] {
				t.Errorf("fScope.Child(1).Child(0) = %v; want = %v", c, infos.Scopes[tag])
			}

			tag2 := elements[1].Body[0].(*ast.OpenTag)
			if infos.Scopes[tag2] == nil {
				t.Error("infos.Scopes[tag2] = nil")
			}
			if c := fScope.Child(2).Child(1).Child(0); c != infos.Scopes[tag2] {
				t.Errorf("fScope.Child(2).Child(1).Child(0) = %v; want = %v", c, infos.Scopes[tag2])
			}

			if infos.Scopes[elements[1]].Lookup("a1") == nil {
				t.Errorf(`infos.Scopes[elements[1]].Lookup("a1") == nil`)
			}
			if infos.Scopes[elements[2]].Lookup("a2") == nil {
				t.Errorf(`infos.Scopes[elements[2]].Lookup("a2") == nil`)
			}
		})
	}
}

func TestTgoScopesInvalid(t *testing.T) {
	const src = `package test

import "github.com/mateusz834/tgo"

func test(tgo.Ctx) error {
	<br
		{
			<br>
			<div
				"test"
			></div>
		}
	>
	return nil
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.tgo", src, parser.SkipObjectResolution|parser.ParseTgo)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Importer: defaultImporter(fset),
		Error:    func(err error) {},
	}
	infos := Info{Scopes: make(map[ast.Node]*Scope)}
	_, err = cfg.Check("test", fset, []*ast.File{f}, &infos)
	t.Log(err)
	if err == nil {
		t.Fatal("Check() did not fail with an error")
	}

	tag := f.Decls[1].(*ast.FuncDecl).Body.List[0].(*ast.OpenTag)
	tag2 := tag.Body[0].(*ast.BlockStmt).List[0].(*ast.OpenTag)
	element := tag.Body[0].(*ast.BlockStmt).List[1].(*ast.Element)

	if infos.Scopes[tag] == nil {
		t.Errorf("infos.Scopes[tag] = <nil>")
	}
	if infos.Scopes[tag2] == nil {
		t.Errorf("infos.Scopes[tag2] = <nil>")
	}
	if infos.Scopes[element] == nil {
		t.Errorf("infos.Scopes[element] = <nil>")
	}
	if infos.Scopes[element.OpenTag] == nil {
		t.Errorf("infos.Scopes[element.OpenTag] = <nil>")
	}
}

// This test checks that no type information is recorded for BasicLit in non-tgo mode,
// we re-use such syntax for *ast.Text in tgo mode. Test just as an alert if something
// changes upstream, so we know.
func TestTgoBasicLit(t *testing.T) {
	const src = `package test
func test() {
	"test"
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	basicLit := f.Decls[0].(*ast.FuncDecl).Body.List[0].(*ast.ExprStmt).X.(*ast.BasicLit)

	infos := Info{Types: map[ast.Expr]TypeAndValue{}}
	cfg := Config{}
	_, err = cfg.Check("test", fset, []*ast.File{f}, &infos)
	if err == nil {
		t.Fatal("Check() did not fail with an error")
	}

	v, ok := infos.Types[basicLit]
	if ok {
		t.Fatalf("infos.Types[basicLit] = %v; want = <nil>", v)
	}
}

func TestTgoInvalidSyntaxTreeTemplateLiteral(t *testing.T) {
	cases := []struct {
		name string
		lit  ast.TemplateLiteral
		errs []string
	}{
		{
			name: "zero val",
			lit:  ast.TemplateLiteral{},
			errs: []string{
				"(*ast.TemplateLiteral).Parts is empty",
				"(*ast.TemplateLiteral).Strings is empty",
			},
		},
		{
			name: "zero parts",
			lit:  ast.TemplateLiteral{Strings: []string{`"`, `"`}},
			errs: []string{
				"(*ast.TemplateLiteral).Parts is empty",
				"(*ast.TemplateLiteral).Strings != len((*ast.TemplateLiteral).Parts)+1",
			},
		},
		{
			name: "zero strings",
			lit:  ast.TemplateLiteral{Parts: []*ast.TemplateLiteralPart{{X: &ast.BasicLit{Kind: token.STRING, Value: `""`}}}},
			errs: []string{
				"(*ast.TemplateLiteral).Strings is empty",
			},
		},
	}

	const src = `package test

import "github.com/mateusz834/tgo"

func test(tgo.Ctx) error {
	"\{1}"
	<div @attr="\{1}"> </div>
	return nil
}
`

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test", src, parser.SkipObjectResolution|parser.ParseComments|parser.ParseTgo)
			if err != nil {
				t.Fatal(err)
			}

			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.TemplateLiteral:
					n.Strings = tt.lit.Strings
					n.Parts = tt.lit.Parts
				}
				return true
			})

			errs := []string{}
			cfg := Config{
				Importer: defaultImporter(fset),
				Error: func(err error) {
					if c := readCode(err.(Error)); c != errors.InvalidSyntaxTree {
						t.Errorf("unexpected error code: %v; want: %v", c, errors.InvalidSyntaxTree)
					}
					errs = append(errs, err.Error())
				},
			}

			_, err = cfg.Check("path", fset, []*ast.File{f}, nil)
			if err == nil {
				t.Fatal("unexpected success")
			}

			want := slices.Concat(tt.errs, tt.errs)
			if len(errs) != len(want) {
				t.Fatalf("got = %v; want = %v", errs, want)
			}

			for i := range want {
				got, want := errs[i], want[i]
				if !strings.Contains(got, want) {
					t.Fatalf("got = %v; want = %v", errs, want)
				}
			}
		})
	}
}

// This test represents the possibly wrong behaviour of the type-checker in terms of errors
// being produced for template literal parts in case the tgo runtime package is not imported directly
// by the file having template literals, but by a other file (in the same package).
// Also see ../internal/types/testdata/tgo/template_literal_invalid_tgo_imported.tgo
// Also see ../internal/types/testdata/tgo/template_literal_invalid_tgo_not_imported.tgo
func TestTgoErrorsRuntimeImportedInDifferentFile(t *testing.T) {
	files := []string{
		`package test; import "github.com/mateusz834/tgo"; var fakeUse tgo.Ctx `,
		`package test; func test() { "\{1.1}" }`,
	}
	for i := range 2 {
		name := "normal"
		files := slices.Clone(files)
		if i == 1 {
			name = "reverse"
			slices.Reverse(files) // order of files should not change the errors.
		}
		t.Run(name, func(t *testing.T) {
			parsedFiles := []*ast.File{}
			fset := token.NewFileSet()
			for i, src := range files {
				f, err := parser.ParseFile(fset, fmt.Sprintf("test%v.tgo", i), src, parser.SkipObjectResolution|parser.ParseComments|parser.ParseTgo)
				if err != nil {
					t.Fatal(err)
				}
				parsedFiles = append(parsedFiles, f)
			}

			errs := []string{}
			cfg := Config{
				Importer: defaultImporter(fset),
				Error: func(err error) {
					errs = append(errs, err.(Error).Msg)
				},
			}

			_, err := cfg.Check("path", fset, parsedFiles, nil)
			if err == nil {
				t.Fatal("unexpected success")
			}

			wantErrs := []string{
				`template literal is not allowed inside a non-tgo function`,
				`float64 does not satisfy tgo.DynamicWriteAllowed (float64 missing in string | github.com/mateusz834/tgo.UnsafeHTML | int | uint | rune)`,
			}

			if !slices.Equal(wantErrs, errs) {
				t.Fatalf("got errors = %v; want = %v", errs, wantErrs)
			}
		})
	}
}

// This test makes sure that even-though we don't have the tgo runtime imported,
// the template parts parts are still type-checked.
func TestTgoRuntimeNotImportedTemplateLitErrors(t *testing.T) {
	const src = `package test

import "math"

func test() {
	"\{1.1} \{1} \{"str"} \{func () error { var err, unused error; return err }()}"
	"\{math.MaxUint}" // This would have cause an overflow error if tgo runtime was imported (int is not inferred).
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file.tgo", src, parser.SkipObjectResolution|parser.ParseComments|parser.ParseTgo)
	if err != nil {
		t.Fatal(err)
	}

	errs := []string{}
	cfg := Config{
		Importer: defaultImporter(fset),
		Error: func(err error) {
			errs = append(errs, err.(Error).Msg)
		},
	}

	infos := &Info{
		Types: map[ast.Expr]TypeAndValue{},
		Defs:  map[*ast.Ident]Object{},
	}
	_, err = cfg.Check("path", fset, []*ast.File{f}, infos)
	if err == nil {
		t.Fatal("unexpected success")
	}

	wantErrs := []string{
		"template literal is not allowed inside a non-tgo function",
		"declared and not used: unused",
		"template literal is not allowed inside a non-tgo function",
	}
	if !slices.Equal(wantErrs, errs) {
		t.Fatalf("got errors = %v; want = %v", errs, wantErrs)
	}

	funcBody := f.Decls[1].(*ast.FuncDecl).Body.List
	templateLit1 := funcBody[0].(*ast.TemplateLiteral)
	templateLit2 := funcBody[1].(*ast.TemplateLiteral)

	// If the tgo runtime had been imported, all of these would be typed.
	wantTypes := []Type{Typ[UntypedFloat], Typ[UntypedInt], Typ[UntypedString], Universe.Lookup("error").Type()}

	for i, want := range wantTypes {
		expr := templateLit1.Parts[i].X
		got := infos.Types[expr]
		if got.Type != want {
			t.Errorf("infos.Types[templateLit1.Parts[%v].X].Type = %v; want = %v", i, got, want)
		}
	}

	got := infos.Types[templateLit2.Parts[0].X]
	if got.Type != Typ[UntypedInt] {
		t.Errorf("infos.Types[templateLit2.Parts[0].X].Type = %v; want = %v", got, Typ[UntypedInt])
	}

	funcInLastPart := templateLit1.Parts[3].X.(*ast.CallExpr).Fun.(*ast.FuncLit).Body
	names := funcInLastPart.List[0].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names
	if len(names) == 0 {
		t.Errorf("len(names) == 0")
	}
	for _, name := range names {
		if infos.Defs[name] == nil {
			t.Errorf("infos.Defs[%v] = nil", name)
		}
	}
}

// As of writing of this test i am not entirely sure how in go/types
// the "incremental" API is expected to work: see https://go.dev/issue/20124
// But this test just proves that the tgo runtime is kept in the checker and
// next calls to Files() have the tgo runtime avail.
// This is related to [TestTgoErrorsRuntimeImportedInDifferentFile] just so that the behaviour is consistent.
func TestTgoCheckerReuseTemplateLit(t *testing.T) {
	errs := []string{}
	cfg := Config{
		Importer: defaultImporter(fset),
		Error: func(err error) {
			errs = append(errs, err.(Error).Msg)
		},
	}

	fset := token.NewFileSet()
	c := NewChecker(&cfg, fset, NewPackage("", ""), nil)

	const src = `package test; import "github.com/mateusz834/tgo"; var fakeUse tgo.Ctx`
	f, err := parser.ParseFile(fset, "file.go", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	const src2 = `package test; func test2() { "\{1.1}" }`
	f2, err := parser.ParseFile(fset, "file2.go", src2, parser.SkipObjectResolution|parser.ParseComments|parser.ParseTgo)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Files([]*ast.File{f}); err != nil {
		t.Errorf("c.Files(f) = %v; want = <nil>", err)
	}

	if err := c.Files([]*ast.File{f2}); err == nil {
		t.Errorf("c.Files(f2) = <nil>")
	}

	wantErrs := []string{
		"template literal is not allowed inside a non-tgo function",
		"float64 does not satisfy tgo.DynamicWriteAllowed (float64 missing in string | github.com/mateusz834/tgo.UnsafeHTML | int | uint | rune)",
	}
	if !slices.Equal(wantErrs, errs) {
		t.Fatalf("got errors = %v; want = %v", errs, wantErrs)
	}
}

// This test makes sure that the "incremental" API, preserves imports
// between calls to Files, as we in the tgo-mode compare few types
// based on the pointer identity.
func TestTgoCheckerReuse(t *testing.T) {
	errs := []string{}
	cfg := Config{
		Importer: &tgoImportedOnceTestImporter{t: t, i: defaultImporter(fset).(ImporterFrom)},
		Error: func(err error) {
			errs = append(errs, err.(Error).Msg)
		},
	}

	fset := token.NewFileSet()
	c := NewChecker(&cfg, fset, NewPackage("", ""), nil)

	const src = `package test; import "github.com/mateusz834/tgo"; var fakeUse tgo.Ctx`
	f, err := parser.ParseFile(fset, "file.go", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	const src2 = `package test; import "github.com/mateusz834/tgo"; func test2(tgo.Ctx) tgo.Error { "text"; return nil }`
	f2, err := parser.ParseFile(fset, "file2.go", src2, parser.SkipObjectResolution|parser.ParseComments|parser.ParseTgo)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Files([]*ast.File{f}); err != nil {
		t.Errorf("c.Files(f) = %v; want = <nil>", err)
	}

	if err := c.Files([]*ast.File{f2}); err != nil {
		t.Errorf("c.Files(f) = %v; want = <nil>", err)
	}
}

type tgoImportedOnceTestImporter struct {
	t           *testing.T
	i           ImporterFrom
	tgoImported bool
}

func (f *tgoImportedOnceTestImporter) handleImport(path string) {
	if f.tgoImported {
		f.t.Errorf("tgo package imported twice")
	}
	if path == "github.com/mateusz834/tgo" {
		f.tgoImported = true
	}
}

func (f *tgoImportedOnceTestImporter) Import(path string) (*Package, error) {
	f.handleImport(path)
	return f.i.Import(path)
}

func (f *tgoImportedOnceTestImporter) ImportFrom(path, dir string, mode ImportMode) (*Package, error) {
	f.handleImport(path)
	return f.i.ImportFrom(path, dir, mode)
}
