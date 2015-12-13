package stream

import (
	"errors"
	"fmt"

	"github.com/skriptble/nine/element"
)

// ElementMux is a stream element multiplexer. It matches elements based on the
// namespace and tag and calls the handler that matches.
//
// TODO: Should there be fuzzy matching? e.g. be able to match a namespace
// handler if there is not handler for both the namespace and tag?
type ElementMux struct {
	Tag   string
	Space string
	FSM   ElementHandler

	m   []elementEntry
	err error
}

type elementEntry struct {
	space, tag string
	h          ElementHandler
}

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
		em.err = errors.New("space and tag cannot be empty")
		return em
	}
	if h == nil {
		em.err = errors.New("ElementHandler cannot be nil")
		return em
	}
	for _, entry := range em.m {
		if entry.space == space && entry.tag == tag {
			em.err = fmt.Errorf("stream: multiple registrations for %s:%s", space, tag)
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
	h := em.Handler(el.Space, el.Tag)
	return h.HandleElement(el, p)
}

// Handler returns the ElementHandler for the given space and tag pair. Handler
// will always return a non-nil ElementHandler.
func (em ElementMux) Handler(space, tag string) ElementHandler {
	for _, entry := range em.m {
		if space == entry.space && tag == entry.tag {
			return entry.h
		}
	}
	Trace.Printf("No handlers for %s:%s", space, tag)
	return UnsupportedStanza{}
}

// ElementHandler is implemented by types that can process elements. If the
// handler modifies the properties it should return those properties. It should
// return any elements that should be written to the stream the element came
// from.
type ElementHandler interface {
	HandleElement(element.Element, Properties) ([]element.Element, Properties)
}

func (eh ElementMux) Match(el element.Element) bool {
	if el.Space != eh.Space || el.Tag != eh.Tag {
		return false
	}
	return true
}