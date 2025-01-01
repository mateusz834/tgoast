package printer

import (
	"github.com/tgo-lang/lang/ast"
	"github.com/tgo-lang/lang/token"
)

func commentGroupBetween(c *ast.CommentGroup, start, end token.Pos) bool {
	return c.Pos() > start && c.End()-1 < end
}

func isEmptyBody(body []ast.Stmt) bool {
	emptyBody := true
	for _, v := range body {
		if _, ok := v.(*ast.EmptyStmt); !ok {
			emptyBody = false
			break
		}
	}
	return emptyBody
}

func (p *printer) tagForceNewline(tagOpenPos, nameStartPos, nameEndPos, tagClosePos token.Pos, body []ast.Stmt) bool {
	forceNewline := p.lineFor(tagOpenPos) != p.lineFor(nameStartPos)
	nameEndPos--

	if c := p.comment; c != nil && !forceNewline && isEmptyBody(body) {
		off := 0
		if !commentGroupBetween(c, nameEndPos, tagClosePos) && p.cindex < len(p.comments) {
			c = p.comments[p.cindex]
			off = 1
		}
		if commentGroupBetween(c, nameEndPos, tagClosePos) {
			hasNext := false
			if p.cindex+off < len(p.comments) {
				hasNext = commentGroupBetween(p.comments[p.cindex+off], nameEndPos, tagClosePos)
			}
			if !hasNext && p.lineFor(c.Pos()) == p.lineFor(nameStartPos) && p.commentsHaveNewline(c.List) {
				forceNewline = true
			}
		}
	}
	return forceNewline
}

func (p *printer) elementBlockStmt(b *ast.ElementBlockStmt) {
	p.opentag(b.OpenTag)
	indent := 1
	oneline := false
	if p.isOneline(b) {
		indent = 0
		oneline = true
	}
	p.stmtList(b.Body, indent, true, oneline)
	if oneline {
		p.print(noExtraLinebreak)
	} else {
		p.linebreak(p.lineFor(b.EndTag.Pos()), 1, ignore, true)
	}

	p.endtag(b.EndTag)
	if oneline {
		p.print(noExtraLinebreak)
	}
}

func (p *printer) opentag(b *ast.OpenTag) {
	p.setPos(b.OpenPos)
	p.print(token.LSS)

	forceNewline := p.tagForceNewline(b.OpenPos, b.Name.NamePos, b.Name.End(), b.ClosePos, b.Body)

	if forceNewline {
		p.print(indent)
		p.linebreak(p.lineFor(b.Name.NamePos), 1, ignore, false)
	}
	p.setPos(b.Name.NamePos)
	p.print(b.Name)
	if forceNewline && isEmptyBody(b.Body) {
		p.linebreak(p.lineFor(b.Name.NamePos), 1, ignore, false)
	}

	if !forceNewline {
		p.print(indent)
	}

	beforeStmtsLine := p.out.Line
	p.stmtList(b.Body, -1, true, false)
	if beforeStmtsLine != p.out.Line {
		p.linebreak(p.lineFor(b.ClosePos), 1, ignore, false)
	}

	p.print(unindent)

	p.setPos(b.ClosePos)

	p.inStartTag = true
	p.tagStartLine = p.lineFor(b.OpenPos)
	p.tagEndLine = p.lineFor(b.ClosePos)
	p.print(token.GTR)
	p.inStartTag = false
}

func (p *printer) endtag(b *ast.EndTag) {
	p.setPos(b.OpenPos)
	p.print(token.END_TAG)

	forceNewline := p.tagForceNewline(b.OpenPos, b.Name.NamePos, b.Name.End(), b.ClosePos, nil)

	if forceNewline {
		p.print(indent)
		p.linebreak(p.lineFor(b.Name.NamePos), 1, ignore, false)
	}

	p.setPos(b.Name.NamePos)
	p.print(b.Name)
	if forceNewline {
		p.print(unindent)
		p.linebreak(p.lineFor(b.ClosePos), 1, ignore, false)
	}

	p.print(indent, unindent)
	p.setPos(b.ClosePos)

	p.inEndTag = true
	p.tagStartLine = p.lineFor(b.OpenPos)
	p.tagEndLine = p.lineFor(b.ClosePos)
	p.print(token.GTR)
	p.inEndTag = false
}

func (p *printer) attr(a *ast.AttributeStmt) {
	p.setPos(a.StartPos)
	p.print(token.AT)

	p.setPos(a.AttrName.Pos())
	p.print(a.AttrName)

	if a.AssignPos != token.NoPos {
		p.setPos(a.AssignPos)
		p.print(token.ASSIGN)
		p.setPos(a.Value.Pos())
		p.expr(a.Value)
	}
}

func (p *printer) templateLiteralExpr(x *ast.TemplateLiteralExpr) {
	p.setPos(x.OpenPos)
	p.print(x.Strings[0])
	for i := range x.Parts {
		p.print("\\", token.LBRACE)
		p.setPos(x.Parts[i].LBrace)
		p.expr(stripParensAlways(x.Parts[i].X))
		p.setPos(x.Parts[i].RBrace)
		if p.mode&noExtraLinebreak != 0 || p.mode&noExtraBlank != 0 {
			panic("unreachable")
		}
		p.print(noExtraLinebreak|noExtraBlank, token.RBRACE, noExtraLinebreak|noExtraBlank)
		p.print(x.Strings[i+1])
	}
}

func (p *printer) isOneline(b *ast.ElementBlockStmt) bool {
	if p.lineFor(b.OpenTag.Pos()) != p.lineFor(b.EndTag.End()) {
		return false
	}

	oneline := len(b.OpenTag.Body) == 0

	var checkList func(list []ast.Stmt)
	checkList = func(list []ast.Stmt) {
		hasStringNodes := false
		hasTagNodes := false
		for _, v := range list {
			switch v := v.(type) {
			case *ast.ExprStmt:
				switch v := v.X.(type) {
				case *ast.BasicLit:
					if v.Kind == token.STRING {
						hasStringNodes = true
						continue
					}
				case *ast.TemplateLiteralExpr:
					hasStringNodes = true
					continue
				}
			case *ast.OpenTag:
				if len(v.Body) == 0 {
					hasTagNodes = true
					continue
				}
			case *ast.EndTag:
				hasTagNodes = true
				continue
			case *ast.ElementBlockStmt:
				hasTagNodes = true
				if len(v.OpenTag.Body) == 0 {
					checkList(v.Body)
					continue
				}
			}
			oneline = false
			return
		}

		if hasTagNodes && hasStringNodes {
			oneline = false
		}
	}

	checkList(b.Body)

	return oneline
}
