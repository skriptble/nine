package stanza

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
)

var IQEmpty = IQ{}

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

func TransformIQ(el element.Element) IQ {
	if el.Tag != "iq" {
		return IQEmpty
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

	return iq
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

func (iq IQ) String() string {
	b := iq.TransformElement().WriteBytes()
	return string(b)
}

func NewIQResult(to, from jid.JID, id string, iqType IQType) IQ {
	s := NewStanza(to, from, id, string(iqType))
	return IQ{}.LoadStanza(s)
}

func NewIQError(iq IQ, err element.Element) IQ {
	to := jid.New(iq.From)
	from := jid.New(iq.To)

	s := NewStanza(to, from, iq.ID, string(IQError))
	s.Children = append(s.Children, err)

	return IQ{}.LoadStanza(s)
}
