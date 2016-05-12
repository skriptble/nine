package stream

import (
	"errors"
	"fmt"

	"github.com/skriptble/nine/element"
)

// ErrNilFeatureHandler is returned from a FeaturesMux.Handle call if the
// provided FeatureHandler is nil
var ErrNilFeatureHandler = errors.New("FeatureHandler cannot be nil")

// ErrSpaceTagEmpty is returned from a FeaturesMux.Handle call if the space or
// tag is empty.
var ErrSpaceTagEmpty = errors.New("space or tag cannot be empty")

// FeaturesMux is a stream features multiplexer. It matches elements based on
// the namespace and tag of the child elements. When multiple children have
// matching handlers the handler with the higher weight is used.
// If two handlers have the same weight, the last one to be registered will
// be choosen.
type FeaturesMux struct {
	handlers []featuresEntry
	err      error
}

type featuresEntry struct {
	space, tag string
	weight     int
	h          FeatureHandler
}

// FeatureHandler is implemented by types that can process stream feature
// elements. If the handler modifies the properties it should return them.
// It should return any elements that should be written to the stream the
// stanza came from.
type FeatureHandler interface {
	HandleFeature(element.Element, Properties) ([]element.Element, Properties)
}

func NewFeaturesMux() FeaturesMux {
	return FeaturesMux{}
}

// Handle registers the FeatureHanlder for the given space, tag, and weight.
// Handlers registered with a negative weight will never be called.
//
// This method is meant to be chained. If an error occurs all following calls
// to Handle are skipped. The error can be retrieved by calling Err().
//
//		fm := NewFeaturesMux().
//					Handle(...).
//					Handle(...).
//					Handle(...)
//		if fm.Err() != nil {
//			// handle error
//			panic(fm.Err())
//		}
func (fm FeaturesMux) Handle(space, tag string, weight int, h FeatureHandler) FeaturesMux {
	if fm.err != nil {
		return fm
	}
	if space == "" || tag == "" {
		fm.err = ErrSpaceTagEmpty
		return fm
	}
	if h == nil {
		fm.err = ErrNilFeatureHandler
		return fm
	}
	for _, entry := range fm.handlers {
		if entry.space == space && entry.tag == tag {
			fm.err = fmt.Errorf("Multiple registrations for tag <%s:%s>", space, tag)
			return fm
		}
	}
	entry := featuresEntry{space: space, tag: tag, weight: weight, h: h}
	fm.handlers = append(fm.handlers, entry)
	return fm
}

// Err returns an error set on the FeaturesMux. This method is usually called
// ater a call to a chain of Handle() methods.
func (fm FeaturesMux) Err() error {
	return fm.err
}

// Handler returns the FeatureHandler for a given space and tag combination.
// Handler will always return a non-nil FeatureHandler.
func (fm FeaturesMux) Handler(els []element.Element) (fh FeatureHandler, elem element.Element) {
	var current featuresEntry
	var space, tag string
	fh = NoOpFeatureHandler{}
	for _, el := range els {
		space, tag = el.Space, el.Tag
		for _, entry := range fm.handlers {
			if space == entry.space && tag == entry.tag && current.weight < entry.weight {
				current = entry
				fh, elem = entry.h, el
			}
		}
	}
	return
}

// HandleElement handles the stream:features element. It finds the
// FeatureHandler to call for the given feature children elements invokes it.
func (fm FeaturesMux) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	children := el.ChildElements()
	h, elem := fm.Handler(children)
	return h.HandleFeature(elem, p)
}

// NoOpFeatureHandler is a FeatureHanlder implementation which does nothing. It
// is the default feature handler returned when no other matching
// FeatureHandler is found.
type NoOpFeatureHandler struct{}

// HandleFeature implements the FeatureHandler interface for
// NoOpFeatureHandler. It is a no-op.
func (nofh NoOpFeatureHandler) HandleFeature(_ element.Element, p Properties) ([]element.Element, Properties) {
	return []element.Element{}, p
}
