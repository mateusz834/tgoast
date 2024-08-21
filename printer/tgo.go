package printer

import (
	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/token"
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

func (p *printer) opentag(b *ast.OpenTagStmt) {
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
	p.stmtList(b.Body, -1, true)
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

	// TODO(nmateusz834): void elements
	p.print(indent)
}

func (p *printer) endtag(b *ast.EndTagStmt) {
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
		p.print("\\{")
		p.expr0(x.Parts[i], 2)
		p.setPos(x.End())
		p.print("}")
		p.print(x.Strings[i+1])
	}
}

func (p *printer) oneLineTag(list []ast.Stmt) bool {
	deep := 0
	startPos := token.NoPos
	for i, v := range list {
		if _, ok := v.(*ast.OpenTagStmt); ok {
			// TODO(mateusz834): void elements
			deep++
			startPos = v.Pos()
		}
		if _, ok := v.(*ast.EndTagStmt); ok {
			if deep--; deep == 0 {
				return p.lineFor(startPos) == p.lineFor(v.End()) &&
					!p.willHaveNewLine(list[0].(*ast.OpenTagStmt), list[1:i])
			}
		}
	}

	panic("unreachable")
}

func (p *printer) willHaveNewLine(o *ast.OpenTagStmt, list []ast.Stmt) bool {
	if v, ok := p.hasNewline[o]; ok {
		return v
	}

	cfg := Config{Mode: RawFormat}
	var counter sizeCounter
	if err := cfg.fprint(&counter, p.fset, list, p.nodeSizes, p.hasNewline); err != nil {
		return true
	}

	var counter2 sizeCounter
	if err := cfg.fprint(&counter2, p.fset, o, p.nodeSizes, p.hasNewline); err != nil {
		return true
	}

	forceMultiLine := false
	for _, v := range list {
		switch v := v.(type) {
		case *ast.OpenTagStmt, *ast.EndTagStmt:
			continue
		case *ast.ExprStmt:
			switch v := v.X.(type) {
			case *ast.BasicLit:
				if v.Kind == token.STRING {
					continue
				}
			case *ast.TemplateLiteralExpr:
				continue
			}
		}
		forceMultiLine = true
	}

	p.hasNewline[o] = counter.hasNewline || counter2.hasNewline || forceMultiLine
	return counter.hasNewline || counter2.hasNewline || forceMultiLine
}
