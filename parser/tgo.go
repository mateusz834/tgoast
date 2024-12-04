package parser

import (
	"slices"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/token"
)

func (p *parser) nextTgoTemplate() {
	if p.tok == token.STRING_TEMPLATE {
		pos := p.pos
		p.templateLit = append(p.templateLit, nil)
		i := len(p.templateLit) - 1
		p.templateLit[i] = p.parseTemplateLiteral()
		p.tok = token.STRING_TEMPLATE
		p.pos = pos
		p.lit = ""
	}
}

func (p *parser) parseTgoOpenTag() *ast.OpenTag {
	if p.tok != token.LSS {
		panic("unreachable")
	}
	openPos := p.pos
	p.next()

	if p.tok == token.RBRACE || p.tok == token.CASE || p.tok == token.DEFAULT ||
		p.tok == token.END_TAG || p.tok == token.LSS {
		p.errorExpected(p.pos, "'"+token.IDENT.String()+"'")
		return &ast.OpenTag{OpenPos: openPos}
	}

	ident := p.parseIdent()

	if p.tok != token.AT && p.tok != token.GTR {
		p.expectSemi()
	}

	if p.tok == token.RBRACE || p.tok == token.END_TAG || p.tok == token.LSS {
		p.errorExpected(p.pos, "'"+token.GTR.String()+"'")
		return &ast.OpenTag{OpenPos: openPos, Name: ident}
	}

	body := p.parseTagStmtList()

	p.scanner.AllowInsertSemiAfterGTR()

	closePos := p.pos
	if p.tok == token.GTR {
		p.next()
	} else {
		closePos = token.NoPos
		p.errorExpected(p.pos, "'"+token.GTR.String()+"'")
		if p.tok != token.RBRACE {
			p.next()
		}
	}

	if p.tok != token.STRING && p.tok != token.STRING_TEMPLATE &&
		p.tok != token.END_TAG && p.tok != token.LSS {
		p.expectSemi()
	}

	return &ast.OpenTag{
		OpenPos:  openPos,
		Name:     ident,
		Body:     body,
		ClosePos: closePos,
	}
}

func (p *parser) parseTgoCloseTag() *ast.EndTag {
	if p.tok != token.END_TAG {
		panic("unreachable")
	}
	openPos := p.pos
	p.next()

	if p.tok == token.RBRACE || p.tok == token.CASE || p.tok == token.DEFAULT ||
		p.tok == token.END_TAG || p.tok == token.LSS {
		p.errorExpected(p.pos, "'"+token.IDENT.String()+"'")
		return &ast.EndTag{OpenPos: openPos}
	}

	ident := p.parseIdent()

	if p.tok != token.AT && p.tok != token.GTR {
		p.expectSemi()
	}

	if p.tok == token.RBRACE || p.tok == token.END_TAG || p.tok == token.LSS {
		p.errorExpected(p.pos, "'"+token.GTR.String()+"'")
		return &ast.EndTag{OpenPos: openPos, Name: ident}
	}

	p.scanner.AllowInsertSemiAfterGTR()
	closePos := p.expect2(token.GTR)
	if p.tok != token.END_TAG && p.tok != token.LSS {
		p.expectSemi()
	}
	return &ast.EndTag{
		OpenPos:  openPos,
		Name:     ident,
		ClosePos: closePos,
	}
}

func unlabel(s ast.Stmt) ast.Stmt {
	for {
		if l, ok := s.(*ast.LabeledStmt); ok {
			s = l.Stmt
			continue
		}
		return s
	}
}

func unlabel2(labeled ast.Stmt) (lastLabeledStmt *ast.LabeledStmt, unlabeled ast.Stmt) {
	unlabeled = labeled
	for {
		if l, ok := unlabeled.(*ast.LabeledStmt); ok {
			unlabeled = l.Stmt
			lastLabeledStmt = l
			continue
		}
		return
	}
}

/*
<div>
	</span>
</div>
*/

func combineElemmentBlocks(list []ast.Stmt) (out []ast.Stmt) {
	type openTag struct {
		openTag int
		body    []ast.Stmt
	}

	openTagDepth := make([]openTag, 0, 16)

	for i, stmt := range list {
		lastLabeledStmt, unlabeledStmt := unlabel2(stmt)
	outer:
		switch unlabeledStmt := unlabeledStmt.(type) {
		case *ast.OpenTag:
			openTagDepth = append(openTagDepth, openTag{openTag: i})
		case *ast.EndTag:
			for i, lastOpenTagData := range slices.Backward(openTagDepth) {
				openTag := list[lastOpenTagData.openTag]
				lastLabeledOpen, unlabeled := unlabel2(openTag)
				unlabeledOpenTag := unlabeled.(*ast.OpenTag)

				if unlabeledOpenTag.Name.Name == unlabeledStmt.Name.Name {
					for j := len(openTagDepth) - 1; j >= i+1; j-- {
						cur := openTagDepth[j]
						prev := &openTagDepth[j-1]
						prev.body = append(prev.body, list[cur.openTag])
						prev.body = append(prev.body, cur.body...)
					}
					lastOpenTagData = openTagDepth[i]
					openTagDepth = openTagDepth[:i]

					if lastLabeledStmt != nil {
						lastLabeledOpen.Stmt = &ast.EmptyStmt{}
						lastOpenTagData.body = append(lastOpenTagData.body, stmt)
					}

					var s ast.Stmt = &ast.ElementBlockStmt{
						OpenTag: unlabeledOpenTag,
						Body:    lastOpenTagData.body,
						EndTag:  unlabeledStmt,
					}

					if lastLabeledOpen != nil {
						lastLabeledOpen.Stmt = s
						s = openTag
					}

					if len(openTagDepth) != 0 {
						last := &openTagDepth[len(openTagDepth)-1]
						last.body = append(last.body, s)
					} else {
						out = append(out, s)
					}

					break outer
				}
			}

			// end tag skipped
			if len(openTagDepth) != 0 {
				last := &openTagDepth[len(openTagDepth)-1]
				last.body = append(last.body, stmt)
			} else {
				out = append(out, stmt)
			}
		default:
			if len(openTagDepth) != 0 {
				last := &openTagDepth[len(openTagDepth)-1]
				last.body = append(last.body, stmt)
			} else {
				out = append(out, stmt)
			}
		}
	}
	return
}

func (p *parser) parseTgoStmt() (s ast.Stmt) {
	switch p.tok {
	case token.LSS:
		return p.parseTgoOpenTag()
	case token.END_TAG:
		return p.parseTgoCloseTag()
	case token.STRING_TEMPLATE:
		lit := p.templateLit[len(p.templateLit)-1]
		p.templateLit = p.templateLit[:len(p.templateLit)-1]
		if lit == nil {
			// TODO: figure out if this can happen
			panic("unreachable")
		}
		p.next()
		p.expectSemiAllowEndTag()
		return &ast.ExprStmt{X: lit}
	case token.AT:
		startPos := p.pos

		p.next()
		ident := p.parseIdent()

		if p.tok == token.ASSIGN {
			assignPos := p.pos

			p.next()

			var val ast.Expr
			if p.tok == token.STRING {
				val = &ast.BasicLit{
					ValuePos: p.pos,
					Kind:     p.tok,
					Value:    p.lit,
				}
				p.next()
			} else if p.tok == token.STRING_TEMPLATE {
				lit := p.templateLit[len(p.templateLit)-1]
				p.templateLit = p.templateLit[:len(p.templateLit)-1]
				if lit == nil {
					panic("unreachable")
				}
				val = lit
				p.next()
			} else {
				p.expect(token.STRING)
			}

			endPos := assignPos
			if val != nil {
				endPos = val.End() - 1
			}

			if p.tok != token.AT && p.tok != token.GTR {
				p.expectSemi()
			}

			return &ast.AttributeStmt{
				StartPos:  startPos,
				AttrName:  ident,
				AssignPos: assignPos,
				Value:     val,
				EndPos:    endPos,
			}
		}

		if p.tok != token.AT && p.tok != token.GTR {
			p.expectSemi()
		}

		return &ast.AttributeStmt{
			StartPos: startPos,
			AttrName: ident,
			EndPos:   ident.End() - 1,
		}
	}

	return nil
}

func (p *parser) parseTagStmtList() (list []ast.Stmt) {
	if p.trace {
		defer un(trace(p, "TagStatementList"))
	}

	for p.tok != token.CASE && p.tok != token.DEFAULT && p.tok != token.GTR && p.tok != token.RBRACE && p.tok != token.EOF {
		list = append(list, p.parseStmt())
	}

	return
}

func (p *parser) parseElementBlockStmtList() (list []ast.Stmt) {
	if p.trace {
		defer un(trace(p, "TagStatementList"))
	}

	for p.tok != token.CASE && p.tok != token.DEFAULT && p.tok != token.END_TAG && p.tok != token.RBRACE && p.tok != token.EOF {
		list = append(list, p.parseStmt())
	}

	return
}

func (p *parser) parseTemplateLiteral() *ast.TemplateLiteralExpr {
	var (
		startPos = p.pos
		strings  = []string{p.lit}
		parts    = []*ast.TemplateLiteralPart{}

		closePos token.Pos
	)

	for {
		lBracePos := token.Pos(int(p.pos) + len(p.lit) + 1)
		p.next()
		parts = append(parts, &ast.TemplateLiteralPart{
			LBrace: lBracePos,
			X:      p.parseExpr(),
			RBrace: p.pos,
		})
		if p.tok != token.RBRACE {
			p.errorExpected(p.pos, "'"+token.RBRACE.String()+"'")
		}
		p.pos, p.tok, p.lit = p.scanner.TemplateLiteralContinue()
		strings = append(strings, p.lit)
		if p.tok == token.STRING {
			closePos = p.pos + token.Pos(len(p.lit)) - 1
			break
		}
	}

	return &ast.TemplateLiteralExpr{
		OpenPos:  startPos,
		Strings:  strings,
		Parts:    parts,
		ClosePos: closePos,
	}
}

func (p *parser) expectSemiAllowEndTag() (comment *ast.CommentGroup) {
	if p.tok != token.END_TAG {
		return p.expectSemi()
	}
	return nil
}
