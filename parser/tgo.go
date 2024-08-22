package parser

import (
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

func (p *parser) parseTgoStmt() (s ast.Stmt) {
	switch p.tok {
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
	case token.LSS, token.END_TAG:
		openPos := p.pos
		closing := p.tok == token.END_TAG
		p.next()

		ident := p.parseIdent()

		if p.tok != token.AT && p.tok != token.GTR {
			p.expectSemi()
		}

		if closing {
			p.scanner.AllowInsertSemiAfterGTR()
			closePos := p.expect2(token.GTR)
			if p.tok != token.STRING && p.tok != token.STRING_TEMPLATE &&
				p.tok != token.END_TAG && p.tok != token.LSS {
				p.expectSemi()
			}
			return &ast.EndTagStmt{
				OpenPos:  openPos,
				Name:     ident,
				ClosePos: closePos,
			}
		}

		// TODO: this might allow tags inside?
		body := p.parseTagStmtList()
		p.scanner.AllowInsertSemiAfterGTR()
		closePos := p.expect2(token.GTR)
		if p.tok != token.STRING && p.tok != token.STRING_TEMPLATE &&
			p.tok != token.END_TAG && p.tok != token.LSS {
			p.expectSemi()
		}

		return &ast.OpenTagStmt{
			OpenPos:  openPos,
			Name:     ident,
			Body:     body,
			ClosePos: closePos,
		}
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

func (p *parser) parseTemplateLiteral() *ast.TemplateLiteralExpr {
	startPos := p.pos
	strings := []string{p.lit}
	parts := []ast.Expr{}

	var closePos token.Pos

	for {
		p.next()
		parts = append(parts, p.parseExpr())
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
