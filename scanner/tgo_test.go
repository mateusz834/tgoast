package scanner

import (
	"strings"
	"testing"

	"github.com/tgo-lang/lang/internal/defaulttgo"
	"github.com/tgo-lang/lang/token"
)

func init() {
	defaulttgo.Enable()
}

func TestTgo(t *testing.T) {
	defaulttgo.ExpectDisabled(t) // This test handles both non-tgo and tgo.

	type tok struct {
		pos token.Pos
		tok token.Token
		lit string
	}

	cases := []struct {
		mode Mode
		src  string
		toks []tok
	}{
		{ScanTgo, "@", []tok{{1, token.AT, ""}, {2, token.EOF, ""}}},
		{ScanTgo, "</", []tok{{1, token.END_TAG, ""}, {3, token.EOF, ""}}},
		{ScanTgo, "</tag", []tok{{1, token.END_TAG, ""}, {3, token.IDENT, "tag"}, {6, token.SEMICOLON, "\n"}, {6, token.EOF, ""}}},

		// In Go "<//' would have been a LSS and a COMMENT, so we preserve this behaviour.
		// In Tgo we require that after "</" there is no additional '/' or '*' to align with Go.
		{ScanTgo | ScanComments, "<//", []tok{{1, token.LSS, ""}, {2, token.COMMENT, "//"}, {4, token.EOF, ""}}},
		{ScanTgo | ScanComments, "</**/", []tok{{1, token.LSS, ""}, {2, token.COMMENT, "/**/"}, {6, token.EOF, ""}}},
		{ScanTgo | ScanComments, "</ //", []tok{{1, token.END_TAG, ""}, {4, token.COMMENT, "//"}, {6, token.EOF, ""}}},
		{ScanTgo | ScanComments, "</ /**/", []tok{{1, token.END_TAG, ""}, {4, token.COMMENT, "/**/"}, {8, token.EOF, ""}}},
		{ScanTgo | ScanComments, "< //", []tok{{1, token.LSS, ""}, {3, token.COMMENT, "//"}, {5, token.EOF, ""}}},
		{ScanTgo | ScanComments, "< /**/", []tok{{1, token.LSS, ""}, {3, token.COMMENT, "/**/"}, {7, token.EOF, ""}}},
		{ScanTgo, "<//", []tok{{1, token.LSS, ""}, {4, token.EOF, ""}}},
		{ScanTgo, "</**/", []tok{{1, token.LSS, ""}, {6, token.EOF, ""}}},
		{ScanTgo, "</ //", []tok{{1, token.END_TAG, ""}, {6, token.EOF, ""}}},
		{ScanTgo, "</ /**/", []tok{{1, token.END_TAG, ""}, {8, token.EOF, ""}}},
		{ScanTgo, "< //", []tok{{1, token.LSS, ""}, {5, token.EOF, ""}}},
		{ScanTgo, "< /**/", []tok{{1, token.LSS, ""}, {7, token.EOF, ""}}},

		{ScanTgo | ScanComments, "</**//", []tok{{1, token.LSS, ""}, {2, token.COMMENT, "/**/"}, {6, token.QUO, ""}, {7, token.EOF, ""}}},
		{ScanTgo, "</**//", []tok{{1, token.LSS, ""}, {6, token.QUO, ""}, {7, token.EOF, ""}}},
	}

	// Generate additional cases for non-[ScanTgo] mode.
	for _, tt := range cases {
		toks := []tok{}
		for _, t := range tt.toks {
			switch t.tok {
			case token.AT:
				toks = append(toks, tok{t.pos, token.ILLEGAL, "@"})
			case token.END_TAG:
				toks = append(toks, tok{t.pos, token.LSS, ""})
				toks = append(toks, tok{t.pos + 1, token.QUO, ""})
			default:
				toks = append(toks, t)
			}
		}
		tt := tt
		tt.toks = toks
		tt.mode = tt.mode &^ ScanTgo // clear [ScanTgo] bit
		cases = append(cases, tt)
	}

	for _, tt := range cases {
		var s Scanner
		fset := token.NewFileSet()
		f := fset.AddFile("test", fset.Base(), len(tt.src))

		var handler func(pos token.Position, msg string)
		s.Init(f, []byte(tt.src), func(pos token.Position, msg string) {
			if handler != nil {
				handler(pos, msg)
				return
			}
			t.Errorf("error handler unexpectedly called (%v, %v)", pos, msg)
		}, tt.mode)

		mode := []string{}
		if tt.mode&ScanComments != 0 {
			mode = append(mode, "ScanComments")
		}
		if tt.mode&ScanTgo != 0 {
			mode = append(mode, "ScanTgo")
		}
		if len(mode) == 0 {
			mode = append(mode, "0")
		}

		wantErrs := 0
		for _, want := range tt.toks {
			handler = nil
			if want.tok == token.ILLEGAL {
				handler = func(pos token.Position, msg string) {
					handler = nil
					if pos != fset.Position(want.pos) || msg != "illegal character U+0040 '@'" {
						t.Errorf("error handler called with (%v, %v); want = (%v, illegal character U+0040 '@')", pos, msg, fset.Position(want.pos))
					}
					wantErrs++
				}
			}

			pos, tok, lit := s.Scan()
			if pos != want.pos || tok != want.tok || lit != want.lit {
				t.Errorf("Scan(%v, %q) = (%v, %v, %q); want = (%v, %v, %q)", strings.Join(mode, "|"), tt.src, pos, tok, lit, want.pos, want.tok, want.lit)
			} else {
				t.Logf("Scan(%v, %q) = (%v, %v, %q)", strings.Join(mode, "|"), tt.src, pos, tok, lit)
			}

			if want.tok == token.ILLEGAL && handler != nil {
				t.Errorf("error handler not called")
			}
		}

		if s.ErrorCount != wantErrs {
			t.Errorf("%q: s.ErrorCount = %v; want = %v", tt.src, s.ErrorCount, wantErrs)
		}
	}
}

func TestNonTgoTemplateLiteral(t *testing.T) {
	defaulttgo.ExpectDisabled(t)

	const src = `"\{a}"`

	fset := token.NewFileSet()
	f := fset.AddFile("test", fset.Base(), len(src))

	var s Scanner
	var errHandlerCalled bool
	s.Init(f, []byte(src), func(pos token.Position, msg string) {
		if errHandlerCalled {
			t.Errorf("error handler unexpectedly called second time (%v, %v)", pos, msg)
		}
		if pos.String() != "test:1:3" || msg != "unknown escape sequence" {
			t.Errorf("error handler called with (%v, %v); want = (test:1:3; unknown escape sequence)", pos, msg)
		}
		errHandlerCalled = true
	}, 0)

	pos, tok, lit := s.Scan()
	if pos != 1 || tok != token.STRING || lit != src {
		t.Errorf("Scan() = (%v, %v, %q); want = (%v, %v, %q)", pos, tok, lit, token.Pos(1), token.STRING, src)
	}
	if !errHandlerCalled {
		t.Errorf("error handler not called")
	}
	if s.ErrorCount != 1 {
		t.Errorf("s.ErrorCount = %v; want = 1", s.ErrorCount)
	}
}

func TestNonTgoPanics(t *testing.T) {
	defaulttgo.ExpectDisabled(t)

	fset := token.NewFileSet()
	f := fset.AddFile("test", fset.Base(), len(""))
	var s Scanner
	s.Init(f, []byte(""), nil, 0)

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("TemplateLiteralContinue did not panic")
			}
			if r, ok := r.(string); ok && strings.Contains(r, "non-ScanTgo mode") {
				return
			}
			panic(r)
		}()
		s.TemplateLiteralContinue()
		t.Fatalf("TemplateLiteralContinue did not panic")
	}()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("AllowInsertSemiAfterGTR did not panic")
			}
			if r, ok := r.(string); ok && strings.Contains(r, "non-ScanTgo mode") {
				return
			}
			panic(r)
		}()
		s.AllowInsertSemiAfterGTR()
		t.Fatalf("AllowInsertSemiAfterGTR did not panic")
	}()
}

func TestAllowInsertSemiAfterGTR(t *testing.T) {
	type tok struct {
		pos token.Pos
		tok token.Token
		lit string

		beforeCallAllowSemi bool
	}

	cases := []struct {
		src  string
		toks []tok
	}{
		{">", []tok{{1, token.GTR, "", false}, {2, token.SEMICOLON, "\n", true}, {2, token.EOF, "", false}}},
		{">\n", []tok{{1, token.GTR, "", false}, {2, token.SEMICOLON, "\n", true}, {3, token.EOF, "", false}}},
		{">", []tok{{1, token.GTR, "", false}, {2, token.EOF, "", false}}},
		{">\n", []tok{{1, token.GTR, "", false}, {3, token.EOF, "", false}}},
		{"<", []tok{{1, token.LSS, "", false}, {2, token.EOF, "", true}}},
		{"<\n", []tok{{1, token.LSS, "", false}, {3, token.EOF, "", true}}},
		{
			">\n>\n",
			[]tok{
				{1, token.GTR, "", false},
				{2, token.SEMICOLON, "\n", true},
				{3, token.GTR, "", false},
				{5, token.EOF, "", false},
			},
		},
	}

	for _, tt := range cases {
		fset := token.NewFileSet()
		f := fset.AddFile("test", fset.Base(), len(tt.src))
		var s Scanner
		s.Init(f, []byte(tt.src), func(pos token.Position, msg string) {
			t.Errorf("error handler unexpectedly called (%v, %v)", pos, msg)
		}, ScanTgo)

		for _, want := range tt.toks {
			if want.beforeCallAllowSemi {
				t.Logf("s.AllowInsertSemiAfterGTR(%q)", tt.src)
				s.AllowInsertSemiAfterGTR()
			}
			pos, tok, lit := s.Scan()
			if pos != want.pos || tok != want.tok || lit != want.lit {
				t.Errorf("Scan(%q) = (%v, %v, %q); want = (%v, %v, %q)", tt.src, pos, tok, lit, want.pos, want.tok, want.lit)
			} else {
				t.Logf("Scan(%q) = (%v, %v, %q)", tt.src, pos, tok, lit)
			}
		}

		if s.ErrorCount != 0 {
			t.Errorf("%q: s.ErrorCount = %v; want = 0", tt.src, s.ErrorCount)
		}
	}
}

func TestTgoTemplateLiteral(t *testing.T) {
	type tok struct {
		pos token.Pos
		tok token.Token
		lit string
	}

	cases := []struct {
		mode Mode
		src  string
		toks []tok
	}{
		{
			mode: ScanTgo,
			src:  `"\{a}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.RBRACE, ""},
				{6, token.STRING, `"`},
				{7, token.SEMICOLON, "\n"},
				{7, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"\{a}"` + "\n",
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.RBRACE, ""},
				{6, token.STRING, `"`},
				{7, token.SEMICOLON, "\n"},
				{8, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"a\{b}c"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"a`},
				{5, token.IDENT, "b"},
				{6, token.RBRACE, ""},
				{7, token.STRING, `c"`},
				{9, token.SEMICOLON, "\n"},
				{9, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"\{a}\{b}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.RBRACE, ""},
				{6, token.STRING_TEMPLATE, ""},
				{8, token.IDENT, "b"},
				{9, token.RBRACE, ""},
				{10, token.STRING, `"`},
				{11, token.SEMICOLON, "\n"},
				{11, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"a\{b}c\{d}e"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"a`},
				{5, token.IDENT, "b"},
				{6, token.RBRACE, ""},
				{7, token.STRING_TEMPLATE, "c"},
				{10, token.IDENT, "d"},
				{11, token.RBRACE, ""},
				{12, token.STRING, `e"`},
				{14, token.SEMICOLON, "\n"},
				{14, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"\{"a"}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.STRING, `"a"`},
				{7, token.RBRACE, ""},
				{8, token.STRING, `"`},
				{9, token.SEMICOLON, "\n"},
				{9, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"a\{"b"}c\{1}d\{"e"}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"a`},
				{5, token.STRING, `"b"`},
				{8, token.RBRACE, ""},
				{9, token.STRING_TEMPLATE, `c`},
				{12, token.INT, "1"},
				{13, token.RBRACE, ""},
				{14, token.STRING_TEMPLATE, `d`},
				{17, token.STRING, `"e"`},
				{20, token.RBRACE, ""},
				{21, token.STRING, `"`},
				{22, token.SEMICOLON, "\n"},
				{22, token.EOF, ""},
			},
		},
		{
			mode: ScanComments | ScanTgo,
			src:  `"\{/*comment*/}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.COMMENT, `/*comment*/`},
				{15, token.RBRACE, ""},
				{16, token.STRING, `"`},
				{17, token.SEMICOLON, "\n"},
				{17, token.EOF, ""},
			},
		},
		{
			mode: ScanComments | ScanTgo,
			src:  `"\{a/*comment*/}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.COMMENT, `/*comment*/`},
				{16, token.RBRACE, ""},
				{17, token.STRING, `"`},
				{18, token.SEMICOLON, "\n"},
				{18, token.EOF, ""},
			},
		},
		{
			mode: ScanComments | ScanTgo,
			src:  `"\{/*comment*/a}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.COMMENT, `/*comment*/`},
				{15, token.IDENT, "a"},
				{16, token.RBRACE, ""},
				{17, token.STRING, `"`},
				{18, token.SEMICOLON, "\n"},
				{18, token.EOF, ""},
			},
		},
		{
			mode: ScanComments | ScanTgo,
			src:  "\"\\{a//comment\n}\"",
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.COMMENT, `//comment`},
				{14, token.SEMICOLON, "\n"},
				{15, token.RBRACE, ""},
				{16, token.STRING, `"`},
				{17, token.SEMICOLON, "\n"},
				{17, token.EOF, ""},
			},
		},
		{
			mode: ScanComments | ScanTgo,
			src:  "\"\\{//comment\na}\"",
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.COMMENT, `//comment`},
				{14, token.IDENT, "a"},
				{15, token.RBRACE, ""},
				{16, token.STRING, `"`},
				{17, token.SEMICOLON, "\n"},
				{17, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  "\"\\{a\n}\"",
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.IDENT, "a"},
				{5, token.SEMICOLON, "\n"},
				{6, token.RBRACE, ""},
				{7, token.STRING, `"`},
				{8, token.SEMICOLON, "\n"},
				{8, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  "\"\\{\n}\"",
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{5, token.RBRACE, ""},
				{6, token.STRING, `"`},
				{7, token.SEMICOLON, "\n"},
				{7, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"\{"\{a}"}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.STRING_TEMPLATE, `"`},
				{7, token.IDENT, `a`},
				{8, token.RBRACE, ""},
				{9, token.STRING, `"`},
				{10, token.RBRACE, ""},
				{11, token.STRING, `"`},
				{12, token.SEMICOLON, "\n"},
				{12, token.EOF, ""},
			},
		},
		{
			mode: ScanTgo,
			src:  `"\{func(){"\{a}"}}"`,
			toks: []tok{
				{1, token.STRING_TEMPLATE, `"`},
				{4, token.FUNC, "func"},
				{8, token.LPAREN, ""},
				{9, token.RPAREN, ""},
				{10, token.LBRACE, ""},
				{11, token.STRING_TEMPLATE, `"`},
				{14, token.IDENT, "a"},
				{15, token.RBRACE, ""},
				{16, token.STRING, `"`},
				{17, token.RBRACE, ""},
				{18, token.RBRACE, ""},
				{19, token.STRING, `"`},
				{20, token.SEMICOLON, "\n"},
				{20, token.EOF, ""},
			},
		},
	}

	// Generate additional cases for non-[ScanComments] mode.
	for _, tt := range cases {
		if tt.mode&ScanComments == 0 {
			continue
		}
		toks := []tok{}
		for _, t := range tt.toks {
			if t.tok == token.COMMENT {
				continue
			}
			toks = append(toks, t)
		}
		tt := tt
		tt.toks = toks
		tt.mode = tt.mode &^ ScanComments // clear [ScanComments] bit
		cases = append(cases, tt)
	}

	for _, tt := range cases {
		fset := token.NewFileSet()
		var s Scanner
		s.Init(fset.AddFile("test", fset.Base(), len(tt.src)), []byte(tt.src), func(pos token.Position, msg string) {
			t.Errorf("error handler unexpectedly called (%v, %v)", pos, msg)
		}, tt.mode)

		depth := []int{}
		for _, want := range tt.toks {
			var (
				pos token.Pos
				tok token.Token
				lit string
			)

			from := ""
			if len(depth) != 0 && depth[len(depth)-1] == 0 {
				depth = depth[:len(depth)-1]
				pos, tok, lit = s.TemplateLiteralContinue()
				if tok != token.STRING && tok != token.STRING_TEMPLATE {
					t.Errorf("TemplateLiteralContinue returned tok = %v; want %v or %v", tok, token.STRING, token.STRING_TEMPLATE)
				}
				from = "TemplateLiteralContinue"
			} else {
				pos, tok, lit = s.Scan()
				from = "Scan"
			}

			switch tok {
			case token.LBRACE:
				if len(depth) != 0 {
					depth[len(depth)-1]++
				}
			case token.RBRACE:
				if len(depth) != 0 {
					depth[len(depth)-1]--
				}
			case token.STRING_TEMPLATE:
				depth = append(depth, 1)
			}

			mode := []string{}
			if tt.mode&ScanComments != 0 {
				mode = append(mode, "ScanComments")
			}
			if tt.mode&ScanTgo != 0 {
				mode = append(mode, "ScanTgo")
			}
			if len(mode) == 0 {
				mode = append(mode, "0")
			}

			if pos != want.pos || tok != want.tok || lit != want.lit {
				t.Errorf("%v(%v, %q) = (%v, %v, %q); want = (%v, %v, %q)", from, strings.Join(mode, "|"), tt.src, pos, tok, lit, want.pos, want.tok, want.lit)
			} else {
				t.Logf("%v(%v, %q) = (%v, %v, %q)", from, strings.Join(mode, "|"), tt.src, pos, tok, lit)
			}

		}

		if s.ErrorCount != 0 {
			t.Errorf("s.ErrorCount = %v; want = 0", s.ErrorCount)
		}
	}
}
