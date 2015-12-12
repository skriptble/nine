package stream

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

// Handler is a generic handler for stanzas. It responds with the service
// unavilable error for every request.
type Handler struct{}

func NewHandler() Handler {
	return Handler{}
}

// HandleIQ implements the stream.IQFSM interface.
func (h Handler) HandleIQ(iq stanza.IQ, props Properties) ([]stanza.Stanza, Properties) {
	var sts []stanza.Stanza
	iq.To = props.From
	iq.From = props.To
	res := stanza.NewIQError(iq, element.Stanza.ServiceUnavailable)

	sts = append(sts, res.TransformStanza())
	return sts, props
}
