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

type UnsupportedStanza struct{}

func (us UnsupportedStanza) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	return []element.Element{element.StreamError.UnsupportedStanzaType}, p
}

type ServiceUnavailable struct{}

func (su ServiceUnavailable) HandleIQ(iq stanza.IQ, p Properties) ([]stanza.Stanza, Properties) {
	res := stanza.NewIQError(iq, element.Stanza.ServiceUnavailable)

	return []stanza.Stanza{res.TransformStanza()}, p
}
