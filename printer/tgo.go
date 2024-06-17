package printer

import (
	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/token"
)

func (p *printer) opentag(b *ast.OpenTagStmt) {
	p.setPos(b.OpenPos)
	p.print(token.LSS)
	p.setPos(b.Name.NamePos)
	p.print(b.Name)

	beforeStmtsLine := p.out.Line

	p.stmtList(b.Body, 1, true)

	if beforeStmtsLine != p.out.Line {
		p.linebreak(p.lineFor(b.ClosePos), 1, ignore, true)
	}

	p.setPos(b.ClosePos)
	p.print(token.GTR)

	// TODO(nmateusz834): void elements
	p.print(indent)
}

func commentGroupBetween(c *ast.CommentGroup, start, end token.Pos) bool {
	return c.Pos() > start && c.End()-1 < end
}

func (p *printer) endtag(b *ast.EndTagStmt) {
	p.inEndTag = true
	defer func() {
		p.inEndTag = false
	}()

	p.setPos(b.OpenPos)
	p.endTagStartLine = p.lineFor(b.OpenPos)
	p.endTagEndLine = p.lineFor(b.ClosePos)

	p.print(token.END_TAG)

	forceNewline := p.lineFor(b.OpenPos) != p.lineFor(b.Name.NamePos)
	if c := p.comment; c != nil && !forceNewline {
		var (
			start, end = b.Name.End() - 1, b.ClosePos
			off        = 0
		)
		if !commentGroupBetween(c, start, end) && p.cindex < len(p.comments) {
			c = p.comments[p.cindex]
			off = 1
		}
		if commentGroupBetween(c, start, end) {
			hasNext := false
			if p.cindex+off < len(p.comments) {
				hasNext = commentGroupBetween(p.comments[p.cindex+off], start, end)
			}
			if !hasNext && p.lineFor(c.Pos()) == p.lineFor(b.Name.Pos()) && p.commentsHaveNewline(c.List) {
				forceNewline = true
			}
		}
	}

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
	p.print(token.GTR)
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
		p.expr(x.Parts[i])
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

	p.hasNewline[o] = counter.hasNewline || counter2.hasNewline
	return counter.hasNewline || counter2.hasNewline
}
