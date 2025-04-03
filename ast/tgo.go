package ast

import (
	"github.com/tgo-lang/lang/token"
)

func walkTgo(v Visitor, node Node) bool {
	switch n := node.(type) {
	case *Element:
		Walk(v, n.OpenTag)
		walkList(v, n.Body)
		Walk(v, n.EndTag)
	case *OpenTag:
		Walk(v, n.Name)
		walkList(v, n.Body)
	case *EndTag:
		Walk(v, n.Name)
	case *Attribute:
		Walk(v, n.AttrName)
		if n.Value != nil {
			Walk(v, n.Value)
		}
	case *TemplateLiteral:
		walkList(v, n.Parts)
	case *TemplateLiteralPart:
		Walk(v, n.X)
	case *Text:
	default:
		return false
	}
	return true
}

type AttrValue interface {
	Node
	attrVal()
}

func (*Text) attrVal()            {}
func (*TemplateLiteral) attrVal() {}

type (
	Element struct {
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

	Attribute struct {
		StartPos  token.Pos // position of the "@" sign
		AttrName  *Ident
		AssignPos token.Pos // position of the "=" sign, might be token.NoPos.
		Value     AttrValue // not nil only when AssignPos != token.NoPos
	}

	TemplateLiteral struct {
		OpenPos  token.Pos // position of the oppening '"'.
		Strings  []string
		Parts    []*TemplateLiteralPart
		ClosePos token.Pos // position of the closing '"'
	}

	TemplateLiteralPart struct {
		LBrace token.Pos
		X      Expr
		RBrace token.Pos
	}

	Text struct {
		StartPos token.Pos
		Text     string
	}
)

func (n *Element) Pos() token.Pos             { return n.OpenTag.Pos() }
func (n *OpenTag) Pos() token.Pos             { return n.OpenPos }
func (n *EndTag) Pos() token.Pos              { return n.OpenPos }
func (n *Attribute) Pos() token.Pos           { return n.StartPos }
func (n *TemplateLiteral) Pos() token.Pos     { return n.OpenPos }
func (n *TemplateLiteralPart) Pos() token.Pos { return n.LBrace }
func (n *Text) Pos() token.Pos                { return n.StartPos }

func (n *Element) End() token.Pos { return n.EndTag.End() }
func (n *OpenTag) End() token.Pos { return n.ClosePos + 1 }
func (n *EndTag) End() token.Pos  { return n.ClosePos + 1 }
func (n *Attribute) End() token.Pos {
	if n.Value != nil {
		return n.Value.End()
	}
	if n.AttrName != nil {
		return n.AttrName.End()
	}
	return n.StartPos + 1
}
func (n *TemplateLiteral) End() token.Pos     { return n.ClosePos + 1 }
func (n *TemplateLiteralPart) End() token.Pos { return n.RBrace + 1 }
func (n *Text) End() token.Pos                { return token.Pos(int(n.StartPos) + len(n.Text)) }

func (*Element) stmtNode()         {}
func (*OpenTag) stmtNode()         {}
func (*EndTag) stmtNode()          {}
func (*Attribute) stmtNode()       {}
func (*TemplateLiteral) stmtNode() {}
func (*Text) stmtNode()            {}
