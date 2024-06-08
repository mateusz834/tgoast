package ast

import "github.com/mateusz834/tgoast/token"

func walkTgo(v Visitor, node Node) bool {
	switch n := node.(type) {
	case *OpenTagStmt:
		v.Visit(n.Name)
		walkStmtList(v, n.Body)
		return true
	case *EndTagStmt:
		v.Visit(n.Name)
		return true
	case *AttributeStmt:
		v.Visit(n.AttrName)
		v.Visit(n.Value)
		return true
	case *TemplateLiteralExpr:
		walkExprList(v, n.Parts)
		return true
	default:
		return false
	}
}

type (
	OpenTagStmt struct {
		OpenPos  token.Pos // position of the "<" sign.
		Name     *Ident
		Body     []Stmt
		ClosePos token.Pos // position of the ">" sign.
	}

	EndTagStmt struct {
		OpenPos  token.Pos // position of the "</" sign.
		Name     *Ident
		ClosePos token.Pos // position of the ">" sign.
	}

	AttributeStmt struct {
		StartPos  token.Pos // positon of the "@" sign
		AttrName  Expr      // *Ident
		AssignPos token.Pos // positon of the "=" sign, might be token.NoPos.
		Value     Expr      // not nil only when AssignPos != token.NoPos
		EndPos    token.Pos
	}
)

func (s *OpenTagStmt) Pos() token.Pos   { return s.OpenPos }
func (s *EndTagStmt) Pos() token.Pos    { return s.OpenPos }
func (s *AttributeStmt) Pos() token.Pos { return s.StartPos }

func (s *OpenTagStmt) End() token.Pos   { return s.ClosePos + 1 }
func (s *EndTagStmt) End() token.Pos    { return s.ClosePos + 1 }
func (s *AttributeStmt) End() token.Pos { return s.EndPos + 1 }

func (s *OpenTagStmt) stmtNode()   {}
func (s *EndTagStmt) stmtNode()    {}
func (s *AttributeStmt) stmtNode() {}

type TemplateLiteralExpr struct {
	OpenPos  token.Pos // positon of the oppening '"'.
	Strings  []string
	Parts    []Expr
	ClosePos token.Pos // position of the closing '"'
}

func (s *TemplateLiteralExpr) Pos() token.Pos { return s.OpenPos }
func (s *TemplateLiteralExpr) End() token.Pos { return s.ClosePos + 1 }
func (s *TemplateLiteralExpr) exprNode()      {}
