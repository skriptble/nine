package stanza

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
)

type Bind struct {
	IQ
}

func NewBindResult(iq IQ, j jid.JID) Bind {
	to := jid.NewJID(iq.To)
	from := jid.NewJID(iq.From)

	s := NewStanza(to, from, iq.ID, string(IQResult))

	result := element.Bind.AddChild(
		element.JID.SetText(j.String()),
	)
	s.AddChild(result)
	iq = IQ{}.LoadStanza(s)

	return Bind{IQ: iq}
}
