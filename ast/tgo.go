package ast

import (
	"github.com/mateusz834/tgoast/token"
)

func walkTgo(v Visitor, node Node) bool {
	switch n := node.(type) {
	case *ElementBlockStmt:
		Walk(v, n.OpenTag)
		walkStmtList(v, n.Body)
		Walk(v, n.EndTag)
		return true
	case *OpenTag:
		Walk(v, n.Name)
		walkStmtList(v, n.Body)
		return true
	case *EndTag:
		Walk(v, n.Name)
		return true
	case *AttributeStmt:
		Walk(v, n.AttrName)
		if n.Value != nil {
			Walk(v, n.Value)
		}
		return true
	case *TemplateLiteralExpr:
		for _, x := range n.Parts {
			Walk(v, x)
		}
		return true
	case *TemplateLiteralPart:
		Walk(v, n.X)
		return true
	default:
		return false
	}
}

type (
	ElementBlockStmt struct {
		OpenTag *OpenTag
		Body    []Stmt
		EndTag  *EndTag
	}

	OpenTag struct {
		OpenPos  token.Pos // position of the "<" sign.
		Name     *Ident
		Body     []Stmt
		ClosePos token.Pos // position of the ">" sign.
	}

	EndTag struct {
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

func (s *OpenTag) Pos() token.Pos          { return s.OpenPos }
func (s *EndTag) Pos() token.Pos           { return s.OpenPos }
func (s *ElementBlockStmt) Pos() token.Pos { return s.OpenTag.Pos() }
func (s *AttributeStmt) Pos() token.Pos    { return s.StartPos }

func (s *OpenTag) End() token.Pos          { return s.ClosePos + 1 }
func (s *EndTag) End() token.Pos           { return s.ClosePos + 1 }
func (s *ElementBlockStmt) End() token.Pos { return s.EndTag.End() }
func (s *AttributeStmt) End() token.Pos    { return s.EndPos + 1 }

func (s *OpenTag) stmtNode()          {}
func (s *EndTag) stmtNode()           {}
func (s *ElementBlockStmt) stmtNode() {}
func (s *AttributeStmt) stmtNode()    {}

type TemplateLiteralExpr struct {
	OpenPos  token.Pos // positon of the oppening '"'.
	Strings  []string
	Parts    []*TemplateLiteralPart
	ClosePos token.Pos // position of the closing '"'
}

func (s *TemplateLiteralExpr) Pos() token.Pos { return s.OpenPos }
func (s *TemplateLiteralExpr) End() token.Pos { return s.ClosePos + 1 }
func (s *TemplateLiteralExpr) exprNode()      {}

type TemplateLiteralPart struct {
	LBrace token.Pos
	X      Expr
	RBrace token.Pos
}

func (s *TemplateLiteralPart) Pos() token.Pos { return s.LBrace }
func (s *TemplateLiteralPart) End() token.Pos { return s.RBrace + 1 }
