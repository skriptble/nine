package stream

import (
	"errors"
	"fmt"

	"github.com/skriptble/nine/element"
)

// ErrEmptySpaceTag is the error set on an ElementMux when Handle is called
// with an empty namespace or empty tag.
var ErrEmptySpaceTag = errors.New("space and tag cannot be empty")

// ErrNilElementHandler is the error set on an ElementMux when Handle is called
// with nil as the parameter for ElementHandler.
var ErrNilElementHandler = errors.New("ElementHandler cannot be nil")

// ErrNilElementMuxerEntry is the error set on an ElementMuxV2 when Handle is
// called with nil as the parameter for ElementMuxerEntry.
var ErrNilElementMuxerEntry = errors.New("ElementMuxerEntry cannot be nil")

// ElementMux is a stream element multiplexer. It matches elements based on the
// namespace and tag and calls the handler that matches.
//
// TODO: Should there be fuzzy matching? e.g. be able to match a namespace
// handler if there is not handler for both the namespace and tag?
type ElementMux struct {
	m              []elementEntry
	err            error
	DefaultHandler ElementHandler
}

type elementEntry struct {
	space, tag string
	h          ElementHandler
}

// NewElementMux returns an initialized ElementMux.
func NewElementMux() ElementMux {
	return ElementMux{}
}

// Handle registers the ElementHandler for the given namespace and tag.
//
// This method is meant to be chained. If an error occurs all following calls
// to Handle are skipped. The error can be retrieved by calling Err().
//
// 		em := NewElementMux().
//				Handle(...).
//				Handle(...).
//				Handle(...)
//		if em.Err() != nil {
//			// handle error
//			panic(em.Err())
//		}
//
// TODO: Determine if a single handler should be able to handle an entire
// namespace.
func (em ElementMux) Handle(space, tag string, h ElementHandler) ElementMux {
	if em.err != nil {
		return em
	}
	if space == "" || tag == "" {
		em.err = ErrEmptySpaceTag
		return em
	}
	if h == nil {
		em.err = ErrNilElementHandler
		return em
	}
	for _, entry := range em.m {
		if entry.space == space && entry.tag == tag {
			em.err = fmt.Errorf("stream: multiple registrations for <%s:%s>", space, tag)
			return em
		}
	}
	entry := elementEntry{space: space, tag: tag, h: h}
	em.m = append(em.m, entry)
	return em
}

// Err returns an error set on the ElementMux. This method is usually called
// after a call to a chain of Handle().
func (em ElementMux) Err() error {
	return em.err
}

// HandleElement dispatches the element to the handler who can handle the space
// and tag combination.
func (em ElementMux) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	h := em.Handler(el)
	return h.HandleElement(el, p)
}

// Handler returns the ElementHandler for the given space and tag pair. Handler
// will always return a non-nil ElementHandler.
func (em ElementMux) Handler(el element.Element) ElementHandler {
	for _, entry := range em.m {
		if el.MatchNamespace(entry.space) && el.Tag == entry.tag {
			return entry.h
		}
	}
	Trace.Printf("No handlers for %s:%s", el.Space, el.Tag)
	if em.DefaultHandler == nil {
		return UnsupportedStanza{}
	}
	return em.DefaultHandler
}

// UnsupportedStanza is an ElementHandler implementation with returns an
// unsupported-stanza-type error for all Elements it handles. This is mainly
// used in the Element multiplexer implementation where it is returned if there
// is no matching handler for a given Element.
type UnsupportedStanza struct{}

// HandleElement returns a stream error of unsupported-stanza-type and sets the
// status bit on the stream to closed.
func (us UnsupportedStanza) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	p.Status = Closed
	return []element.Element{element.StreamError.UnsupportedStanzaType}, p
}

// UnsupportedStanza is an ElementHandler implementation with returns an
// unsupported-stanza-type error for all Elements it handles. This is mainly
// used in the Element multiplexer implementation where it is returned if there
// is no matching handler for a given Element.
type UnsupportedStanzaV2 struct{}

// HandleElement returns a stream error of unsupported-stanza-type and sets the
// status bit on the stream to closed.
func (us UnsupportedStanzaV2) HandleElement(el element.Element) ([]element.Element, StateChange, bool, bool) {
	return []element.Element{element.StreamError.UnsupportedStanzaType}, nil, false, true
}

func (us UnsupportedStanzaV2) Update(_, _ string) {}

// Blackhole is an ElementHandler implementation which does nothing with the
// handled element and returns no elements. This is mainly used as a
// placeholder for message and presence stanzas in nine since the handling of
// those types of stanzas is beyond the scope of RFC6120.
type Blackhole struct{}

// HandleElement does nothing and returns the Properties unchanged.
func (bh Blackhole) HandleElement(_ element.Element) ([]element.Element, StateChange, bool, bool) {
	return []element.Element{}, nil, false, false
}

func (bh Blackhole) Update(_, _ string) {}

// ElementHandler is implemented by types that can process elements. If the
// handler modifies the properties it should return those properties. It should
// return any elements that should be written to the stream the element came
// from.
type ElementHandler interface {
	HandleElement(element.Element, Properties) ([]element.Element, Properties)
}

// ElementHandlerV2 is implemented by types that can process elements. It should
// return any elements that should be written to the stream the element came
// from. If the handler resulted in the need for a stream restart, restart
// should be true. If the handler resulted in the need to close the stream, e.g.
// too many failed authentication attempts, close should be true.
type ElementHandlerV2 interface {
	HandleElement(element.Element) (els []element.Element, restart, close bool)
}

// ElementMuxerEntry is implemented by types that can be used as an entry in
// an ElementMuxer. In addition to being able to handle elements, it can also
// indicate a stream state change by returning a non-empty string as the state
// value when StateChange is called. This will cause the muxer to call the
// Update method on all of the ElementMuxerEntrys it contains, passing it the
// state and payload string returned from StateChange.
type ElementMuxerEntry interface {
	HandleElement(element.Element) (els []element.Element, sc StateChange, restart, close bool)
	MuxerEntry
}

// StateChange allows a muxer entry to update the stream state for a muxer.
// It is meant to be used when the result of a handler has caused a change
// to the state of the stream, e.g. authentication or binding of a
// resource. When state is a non-empty string it results in the
// UpdateHandler method being called for all ElementMuxerEntrys for the
// muxer.
type StateChange func() (state, payload string)

type MuxerEntry interface {
	// Update allows an ElementMuxerEntry to change its internal state in
	// response to a state change initiated by the handling of an element.
	Update(state, payload string)
}
