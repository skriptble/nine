package stanza

import (
	"errors"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/namespace"
)

var ErrNotBindRequest = errors.New("IQ is not a bind request")

type BindResult struct {
	IQ
}

// BindRequest represents an xmpp bind request.
type BindRequest struct {
	Resource string
}

func NewBindResult(iq IQ, j jid.JID) BindResult {
	to := jid.New(iq.To)
	from := jid.New(iq.From)

	s := NewStanza(to, from, iq.ID, string(IQResult))

	result := element.Bind.AddChild(
		element.JID.SetText(j.String()),
	)
	s = s.AddChild(result)
	iq = IQ{}.LoadStanza(s)

	return BindResult{IQ: iq}
}

func TransformBindRequest(iq IQ) (br BindRequest, err error) {
	for _, child := range iq.Children {
		if child.Tag == "bind" && child.Space == namespace.Bind {
			br.Resource = child.SelectElement("resource").Text()
			break
		}
	}

	return
}
