package parser

import (
	"slices"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/token"
)

func (p *parser) combineElemmentBlocks(list []ast.Stmt) (out []ast.Stmt) {
	unlabel := func(labeled ast.Stmt) (lastLabeledStmt *ast.LabeledStmt, unlabeled ast.Stmt) {
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

	type openTagData struct {
		openTagIndex int
		body         []ast.Stmt
	}

	openTagDepth := make([]openTagData, 0, 16)

	appendStmts := func(stmts []ast.Stmt) {
		if len(openTagDepth) != 0 {
			last := &openTagDepth[len(openTagDepth)-1]
			last.body = append(last.body, stmts...)
		} else {
			out = append(out, stmts...)
		}
	}

	last := 0
	for i, stmt := range list {
		lastLabeledStmt, unlabeledStmt := unlabel(stmt)
		switch unlabeledStmt := unlabeledStmt.(type) {
		case *ast.OpenTag:
			appendStmts(list[last:i])
			last = i + 1
			openTagDepth = append(openTagDepth, openTagData{openTagIndex: i})
		case *ast.EndTag:
			for j, lastOpenTagData := range slices.Backward(openTagDepth) {
				openTag := list[lastOpenTagData.openTagIndex]
				lastLabeledOpen, unlabeled := unlabel(openTag)
				unlabeledOpenTag := unlabeled.(*ast.OpenTag)

				if unlabeledOpenTag.Name.Name == unlabeledStmt.Name.Name {
					appendStmts(list[last:i])
					last = i + 1

					body := lastOpenTagData.body

					for _, v := range openTagDepth[j+1:] {
						openTagStmt := list[v.openTagIndex]
						_, openTag := unlabel(openTagStmt)
						name := openTag.(*ast.OpenTag).Name
						if name != nil {
							if name.Name != "br" {
								p.error(openTag.(*ast.OpenTag).OpenPos, "unclosed tag")
							}
						}
						body = append(body, openTagStmt)
						body = append(body, v.body...)
					}

					openTagDepth = openTagDepth[:j]

					if lastLabeledStmt != nil {
						lastLabeledOpen.Stmt = &ast.EmptyStmt{} // TODO:
						body = append(body, stmt)
					}

					var s ast.Stmt = &ast.ElementBlockStmt{
						OpenTag: unlabeledOpenTag,
						Body:    body,
						EndTag:  unlabeledStmt,
					}

					if lastLabeledOpen != nil {
						lastLabeledOpen.Stmt = s
						s = openTag
					}

					appendStmts([]ast.Stmt{s})
					break
				}

				p.error(unlabeledStmt.OpenPos, "unopenned tag")
			}
		}
	}

	if last == 0 {
		out = list
	}

	appendStmts(list[last:])

	return
}

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
