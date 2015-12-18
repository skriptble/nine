package stream

import (
	"errors"
	"fmt"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

// IQMux is a iq stanza multiplexer. It matches IQ stanzas based on the type of
// stanza and the space and tag of the first child element of the stanza.
//
// IQMux will return a NotAuthorized error if an IQ is sent before the stream
// has been authenticated or if an outbound IQ (e.g. has a to address that is
// not the bare jid of the user or the server's domain) is sent before the
// stream has been bound.
//
// IQMux will return an IQ err of service-unavilable is a stanza is sent for
// which there is no matching space, tag, and type.
type IQMux struct {
	handlers []iqEntry
	err      error
}

type iqEntry struct {
	space, tag string
	stanzaType string
	h          IQHandler
}

// NewIQMux initializes and returns an IQ multiplexer.
func NewIQMux() IQMux {
	return IQMux{}
}

// IQHandler is implemented by types that can process IQ stanzas. If the
// handler modifies the properties it should return them. It should return
// any stanzas that should be written to the stream the stanza came from.
//
// TODO: Determine if this interface should return a single IQ stanza. Since IQ
// stanzas are request response, it doesn't make a ton of sense that another
// element would be generated.
type IQHandler interface {
	HandleIQ(stanza.IQ, Properties) ([]stanza.Stanza, Properties)
}

// Handle registers the IQHandler for the given iq type with the first child
// element matching the space and tag.
//
// This method is meant to be chained. If an error occurs all following calls
// to Handle are skipped. The error can be retrieved by calling Err().
//
// 		im := NewIQMux().
//				Handle(...).
//				Handle(...).
//				Handle(...)
//		if im.Err() != nil {
//			// handle error
//			panic(im.Err())
//		}
//
// TODO: Should more fuzzy matching be allowed, such as sending all iq's with
// a given child element to the handler, regardless of the type?
func (im IQMux) Handle(space, tag, stanzaType string, h IQHandler) IQMux {
	if im.err != nil {
		return im
	}
	if space == "" || tag == "" || stanzaType == "" {
		im.err = errors.New("space, tag, or stanzaType cannot be empty")
		return im
	}
	if h == nil {
		im.err = errors.New("IQHandler cannot be nil")
		return im
	}
	for _, entry := range im.handlers {
		if entry.space == space && entry.tag == tag && entry.stanzaType == stanzaType {
			im.err = fmt.Errorf("Multiple registrations for type %s and tag %s:%s", stanzaType, space, tag)
			return im
		}
	}
	entry := iqEntry{space: space, tag: tag, stanzaType: stanzaType, h: h}
	im.handlers = append(im.handlers, entry)
	return im
}

// Err returns an error set on the IQMux. This method is usually called after a
// call to a chain of Handle() calls.
func (im IQMux) Err() error {
	return im.err
}

// Handler returns the IQHandler for the given space, tag, and type combo.
// Handler will always return a non-nil IQHandler.
//
// TODO: Handler should return ServiceUnavailable for get and set IQs and
// should do nothing if the type is result or error.
func (im IQMux) Handler(space, tag, sType string) IQHandler {
	for _, entry := range im.handlers {
		if space == entry.space && tag == entry.tag && sType == entry.stanzaType {
			return entry.h
		}
	}
	return ServiceUnavailable{}
}

// HandleElement dispatches the element (IQ) to the handler who can handle the
// space and tag (of the first child element) combination for the given stanza
// type.
//
// TODO: This method is very receiving entity (server) focused. Redesign this
// to handle the initiating entity usecase.
func (im IQMux) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	iq := stanza.TransformIQ(el)
	// If the stream is not authenticated or bound and is addressed not to the
	// server or the user
	if p.Status&Auth == 0 {
		// TODO: Close the stream by setting the closed bit on status.
		return []element.Element{element.StreamError.NotAuthorized}, p
	}
	if p.Status&Bind == 0 && iq.To != "" {
		if iq.To != p.Domain && iq.To != p.To {
			return []element.Element{element.StreamError.NotAuthorized}, p
		}
	}
	var elems = []element.Element{}
	child := iq.First()
	h := im.Handler(child.Space, child.Tag, iq.Type)
	sts, p := h.HandleIQ(iq, p)
	for _, st := range sts {
		elems = append(elems, st.TransformElement())
	}
	return elems, p
}

// ServiceUnavailable is an IQHandler implementation which returns a Service
// Unavailable stanza for all IQs it handles. This is mainly used in the IQ
// multiplexer implementation where it is returned if there is no matching
// handler for a given IQ.
type ServiceUnavailable struct{}

// HandleIQ handles transforming the given stanza into a service-unavailable
// error iq stanza.
func (su ServiceUnavailable) HandleIQ(iq stanza.IQ, p Properties) ([]stanza.Stanza, Properties) {
	res := stanza.NewIQError(iq, element.Stanza.ServiceUnavailable)

	return []stanza.Stanza{res.TransformStanza()}, p
}
