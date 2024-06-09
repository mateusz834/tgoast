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

	// TODO(mateusz834): void elements
	p.indent++
}

func (p *printer) endtag(b *ast.EndTagStmt) {
	p.indent--

	p.setPos(b.OpenPos)
	p.print(token.END_TAG)
	p.setPos(b.Name.NamePos)
	p.print(b.Name)
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
	p.print(x.Strings[0])
	for i := range x.Parts {
		p.print("\\{")
		p.expr(x.Parts[i])
		p.print("}")
		p.print(x.Strings[i+1])
	}
}

func (p *printer) oneLineTag(list []ast.Stmt) bool {
	deep := 0
	for i, v := range list {
		if _, ok := v.(*ast.OpenTagStmt); ok {
			// TODO(mateusz834): void elements
			deep++
		}
		if _, ok := v.(*ast.EndTagStmt); ok {
			if deep--; deep == 0 {
				return !p.willHaveNewLine(list[0].(*ast.OpenTagStmt), list[1:i])
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

	p.hasNewline[o] = counter.hasNewline
	return counter.hasNewline
}
