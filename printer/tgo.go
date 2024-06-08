package printer

import (
	"slices"

	"github.com/mateusz834/tgoast/ast"
	"github.com/mateusz834/tgoast/token"
)

func (p *printer) endtag(b *ast.EndTagStmt) {
	p.setPos(b.OpenPos)
	p.print(token.END_TAG)
	p.setPos(b.Name.NamePos)
	p.print(b.Name)
	p.print(token.GTR)

	// TODO(mateusz834): void elements
	p.indent++
}

func (p *printer) opentag(b *ast.OpenTagStmt) {
	p.indent--
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

func (p *printer) oneLineTags(list []ast.Stmt) []bool {
	var (
		lineNumbers []int
		out         []bool
	)

	out = append(out, false)

	for _, v := range list {
		switch v := v.(type) {
		case *ast.OpenTagStmt:
			// TODO(mateusz834): void elements.
			lineNumbers = append(lineNumbers, p.lineFor(v.OpenPos))
		case *ast.EndTagStmt:
			last := lineNumbers[len(lineNumbers)-1]
			lineNumbers = lineNumbers[:len(lineNumbers)-1]
			out = append(out, last == p.lineFor(v.ClosePos))
		}
	}

	slices.Reverse(out[1:])
	return out
}
