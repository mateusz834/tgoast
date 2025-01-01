package parser

import (
	"fmt"
	"slices"

	"github.com/tgo-lang/lang/ast"
	"github.com/tgo-lang/lang/token"
)

func unlabelAs[T ast.Stmt](labeled ast.Stmt) (lastLabeledStmt *ast.LabeledStmt, unlabeled T) {
	for {
		if l, ok := labeled.(*ast.LabeledStmt); ok {
			labeled = l.Stmt
			lastLabeledStmt = l
			continue
		}
		return lastLabeledStmt, labeled.(T)
	}
}

func (p *parser) combineElemmentBlocks(list []ast.Stmt) (out []ast.Stmt) {
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
		lastLabeledStmt, unlabeledStmt := unlabelAs[ast.Stmt](stmt)
	outer:
		switch unlabeledStmt := unlabeledStmt.(type) {
		case *ast.OpenTag:
			if !unlabeledStmt.ClosePos.IsValid() {
				continue
			}
			appendStmts(list[last:i])
			last = i + 1
			openTagDepth = append(openTagDepth, openTagData{openTagIndex: i})
		case *ast.EndTag:
			if !unlabeledStmt.ClosePos.IsValid() {
				continue
			}

			for j, lastOpenTagData := range slices.Backward(openTagDepth) {
				openTag := list[lastOpenTagData.openTagIndex]
				lastLabeledOpen, unlabeledOpenTag := unlabelAs[*ast.OpenTag](openTag)

				if unlabeledOpenTag.Name.Name == unlabeledStmt.Name.Name {
					body := lastOpenTagData.body

					for _, v := range openTagDepth[j+1:] {
						openTagStmt := list[v.openTagIndex]
						_, openTag := unlabelAs[*ast.OpenTag](openTagStmt)
						if openTag.Name != nil {
							if openTag.Name.Name != "br" {
								p.error(openTag.OpenPos, "unclosed tag")
							}
						}
						body = append(body, openTagStmt)
						body = append(body, v.body...)
					}

					body = append(body, list[last:i]...)
					last = i + 1

					openTagDepth = openTagDepth[:j]

					if lastLabeledStmt != nil {
						lastLabeledStmt.Stmt = &ast.EmptyStmt{Semicolon: unlabeledStmt.OpenPos, Implicit: true}
						body = append(body, stmt)
					}

					// TODO: if void element, then error.

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
					break outer
				}
			}

			p.error(unlabeledStmt.OpenPos, fmt.Sprintf("unopenend tag: %v", unlabeledStmt.Name.Name))
		}
	}

	if last == 0 {
		out = list
		return
	}

	for _, v := range openTagDepth {
		openTagStmt := list[v.openTagIndex]
		_, openTag := unlabelAs[*ast.OpenTag](openTagStmt)
		if openTag.Name != nil {
			if openTag.Name.Name != "br" {
				p.error(openTag.OpenPos, "unclosed tag")
			}
		}
		out = append(out, openTagStmt)
		out = append(out, v.body...)
	}
	out = append(out, list[last:]...)
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
