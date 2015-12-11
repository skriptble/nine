package stanza

import (
	"errors"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
)

var ErrNotIQ = errors.New("the provided element is not an IQ")

type Stanza struct {
	To   string
	From string
	ID   string
	Type string
	Lang string

	Children   []element.Element
	Data       string
	Tag, Space string
}

func NewStanza(to, from jid.JID, id, sType string) Stanza {
	return Stanza{
		To:   to.String(),
		From: from.String(),
		ID:   id,
		Type: sType,
	}
}

func (s Stanza) AddChild(el element.Element) Stanza {
	s.Children = append(s.Children, el)
	return s
}

func (s Stanza) First() element.Element {
	if len(s.Children) < 1 {
		return element.Element{}
	}
	return s.Children[0]
}

func (s Stanza) SetLang(lang string) Stanza {
	s.Lang = lang
	return s
}

func (s Stanza) SetText(str string) Stanza {
	s.Data = str
	return s
}

func (s Stanza) SetTag(str string) Stanza {
	s.Tag = str
	return s
}

func (s Stanza) SetSpace(str string) Stanza {
	s.Space = str
	return s
}

func (s Stanza) TransformElement() element.Element {
	attrs := []element.Attr{}
	if s.To != "" {
		attrs = append(attrs, element.Attr{Key: "to", Value: s.To})
	}
	if s.From != "" {
		attrs = append(attrs, element.Attr{Key: "from", Value: s.From})
	}
	if s.ID != "" {
		attrs = append(attrs, element.Attr{Key: "id", Value: s.ID})
	}
	if s.Type != "" {
		attrs = append(attrs, element.Attr{Key: "type", Value: s.Type})
	}
	if s.Lang != "" {
		attrs = append(attrs, element.Attr{Key: "lang", Space: "xml", Value: s.Lang})
	}
	el := element.Element{Tag: s.Tag, Space: s.Space, Attr: attrs}
	if s.Data != "" {
		el = el.SetText(s.Data)
	}
	for _, child := range s.Children {
		el.Child = append(el.Child, child)
	}
	return el
}

type Message struct {
	Stanza
}

type Presence struct {
	Stanza
}

type IQ struct {
	Stanza
}

// IQType is the type of an IQ.
type IQType string

const (
	IQGet    IQType = "get"
	IQSet    IQType = "set"
	IQResult IQType = "result"
	IQError  IQType = "error"
)

func IsIQ(el element.Element) bool {
	return el.Tag == "iq"
}

func TransformIQ(el element.Element) (IQ, error) {
	if el.Tag != "iq" {
		return IQ{}, ErrNotIQ
	}
	iq := IQ{}
	iq.To = el.SelectAttrValue("to", "")
	iq.From = el.SelectAttrValue("from", "")
	iq.ID = el.SelectAttrValue("id", "")
	iq.Type = el.SelectAttrValue("type", "")
	iq.Lang = el.SelectAttrValue("xml:lang", "")
	iq.Data = el.Text()

	iq.Children = el.ChildElements()
	iq.Tag, iq.Space = el.Tag, el.Space

	return iq, nil
}

func (iq IQ) TransformElement() element.Element {
	iq.Stanza = iq.Stanza.SetTag("iq")
	return iq.Stanza.TransformElement()
}

func (iq IQ) TransformStanza() Stanza {
	return iq.Stanza
}

func (iq IQ) LoadStanza(s Stanza) IQ {
	s = s.SetTag("iq")
	return IQ{Stanza: s}
}

func NewIQResult(to, from jid.JID, id string, iqType IQType) IQ {
	s := NewStanza(to, from, id, string(iqType))
	return IQ{Stanza: s}
}
