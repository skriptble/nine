package stream

import (
	"fmt"

	"github.com/skriptble/nine/element"
)

// ElementMuxV2 is a stream element multiplexer. It matches elements based on
// the namespace and tag and calls the handler that matches. The muxer also
// handles stream state internally, allowing entries to indicate that the stream
// state has changed and notifying all other handlers, allowing them to update
// themselves. The zero value for an ElementMuxV2 is a valid configuration.
//
// TODO(skriptble): Add the context.Context object here and contextual logging.
type ElementMuxV2 struct {
	m              []elementEntryV2
	err            error
	DefaultHandler ElementMuxerEntry
}

type elementEntryV2 struct {
	space, tag string
	h          ElementMuxerEntry
}

// Handle registers the ElementMuxerEntry for the given namespace and tag.
//
// This method is meant to be chained and creates a new instance of the
// ElementMuxV2 after each call. This results in an immutable ElementMuxerV2.
// If an error occurs all following calls to Handle are skipped. The error can
// be retrieved by calling Err().
//
// 		em := ElementMuxV2{}.
//				Handle(...).
//				Handle(...).
//				Handle(...)
//		if em.Err() != nil {
//			// handle error
//			panic(em.Err())
//		}
//
func (em ElementMuxV2) Handle(space, tag string, h ElementMuxerEntry) ElementMuxV2 {
	if em.err != nil {
		return em
	}
	if space == "" || tag == "" {
		em.err = ErrEmptySpaceTag
		return em
	}
	if h == nil {
		em.err = ErrNilElementMuxerEntry
		return em
	}
	for _, entry := range em.m {
		if entry.space == space && entry.tag == tag {
			em.err = fmt.Errorf("stream: multiple registrations for <%s:%s>", space, tag)
			return em
		}
	}
	entry := elementEntryV2{space: space, tag: tag, h: h}
	em.m = append(em.m, entry)
	return em
}

// Err returns an error set on the ElementMuxV2. This method is usually called
// after a call to a chain of Handle().
func (em ElementMuxV2) Err() error {
	return em.err
}

// HandleElement dispatches the element to the handler who can handle the space
// and tag combination.
func (em ElementMuxV2) HandleElement(el element.Element) (els []element.Element, restart, close bool) {
	var sc StateChange
	h := em.Handler(el)
	els, sc, restart, close = h.HandleElement(el)
	if sc != nil {
		state, payload := sc()
		for _, entry := range em.m {
			entry.h.Update(state, payload)
		}
	}

	return els, restart, close
}

// Handler returns the ElementMuxerEntry for the given space and tag pair.
// Handler will always return a non-nil ElementMuxerEntry.
func (em ElementMuxV2) Handler(el element.Element) ElementMuxerEntry {
	for _, entry := range em.m {
		if el.MatchNamespace(entry.space) && el.Tag == entry.tag {
			return entry.h
		}
	}
	Trace.Printf("Namespace: %v", el.Namespaces)
	Trace.Printf("No handlers for %s:%s", el.Space, el.Tag)
	if em.DefaultHandler == nil {
		return UnsupportedStanzaV2{}
	}
	return em.DefaultHandler
}
