package stanza

import (
	"errors"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
)

var ErrNotIQ = errors.New("the provided element is not an IQ stanza")
var ErrNotPresence = errors.New("the provided element is not a Presence stanza")
var ErrNotMessage = errors.New("the provided element is not a Message stanza")

type Stanza struct {
	To   string
	From string
	ID   string
	Type string
	Lang string

	Children   []element.Element
	Data       string
	Tag, Space string
	Namespaces map[string]string
}

func (s Stanza) String() string {
	b := s.TransformElement().WriteBytes()
	return string(b)
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
	if s.Namespaces != nil {
		for alias, ns := range s.Namespaces {
			// Handle top level namespace
			if alias == "" {
				attrs = append(attrs, element.Attr{Key: "xmlns", Value: ns})
				continue
			}

			attrs = append(attrs, element.Attr{Key: alias, Space: "xmlns", Value: ns})
		}
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

func (m Message) String() string {
	b := m.TransformElement().WriteBytes()
	return string(b)
}

func TransformMessage(el element.Element) (Message, error) {
	if el.Tag != "message" {
		return Message{}, ErrNotMessage
	}
	message := Message{}
	message.To = el.SelectAttrValue("to", "")
	message.From = el.SelectAttrValue("from", "")
	message.ID = el.SelectAttrValue("id", "")
	message.Type = el.SelectAttrValue("type", "")
	message.Lang = el.SelectAttrValue("xml:lang", "")
	message.Data = el.Text()

	message.Children = el.ChildElements()
	message.Tag, message.Space = el.Tag, el.Space

	return message, nil
}

type Presence struct {
	Stanza
}

func (p Presence) String() string {
	b := p.TransformElement().WriteBytes()
	return string(b)
}

func TransformPresence(el element.Element) (Presence, error) {
	if el.Tag != "presence" {
		return Presence{}, ErrNotPresence
	}
	presence := Presence{}
	presence.To = el.SelectAttrValue("to", "")
	presence.From = el.SelectAttrValue("from", "")
	presence.ID = el.SelectAttrValue("id", "")
	presence.Type = el.SelectAttrValue("type", "")
	presence.Lang = el.SelectAttrValue("xml:lang", "")
	presence.Data = el.Text()

	presence.Children = el.ChildElements()
	presence.Tag, presence.Space = el.Tag, el.Space

	return presence, nil
}
