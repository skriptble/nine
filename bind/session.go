package bind

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/stream"
)

// SessionHandler belongs in ten but is a placeholder here for now to get
// clients to connect and hold a connection.
type SessionHandler struct {
}

func NewSessionHandler() SessionHandler {
	return SessionHandler{}
}

func (sh SessionHandler) GenerateFeature(props stream.Properties) stream.Properties {
	if props.Status&stream.Bind != 0 || props.Status&stream.Auth == 0 {
		return props
	}

	props.Features = append(props.Features, element.Session)
	return props
}

func (sh SessionHandler) HandleFeature(props stream.Properties) stream.Properties {
	if props.Status&stream.Bind != 0 || props.Status&stream.Auth == 0 {
		return props
	}

	props.Features = append(props.Features, element.Session)
	return props
}

func (sh SessionHandler) HandleIQ(iq stanza.IQ, props stream.Properties) (
	[]stanza.Stanza, stream.Properties) {
	var sts []stanza.Stanza
	to := jid.New(iq.From)
	from := jid.New(iq.To)
	res := stanza.NewIQResult(to, from, iq.ID, stanza.IQResult)
	sts = append(sts, res.TransformStanza())
	return sts, props
}
