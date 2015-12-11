package features

import (
	"fmt"
	"log"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

// Feature is a state used during feature negotiation. They run inside the
// connection state machine under a FeatureNegotiator. They should return their
// own error state if an error occurs (which will most likely close the stream)
type Feature interface {
	element.Transformer

	// Negotiate should handle the negotiation fo this stream feature. The
	// element provided is the first element read from the stream, this element
	// is used to determine which feature should be run.
	// If the stream returned will replace the current stream.
	// If an error occured and the stream needs to be closed, closeStream
	// should be true. If the feature in the current step need to be retried,
	// retry should be true.
	// restart should be true if the stream needs restarting. This will ensure
	// that stream features are written.
	Negotiate(element.Element, stream.Streamv0) (s stream.Streamv0, restart, closeStream, retry bool)
}

// Step represents a step in the stream negotiation process.
// The key should be the namespaced tag of the first element.
type Step map[string]Feature

// FeatureNegotiator is used to negotiate the features of a stream. It has
// several steps that are executed in order.
type FeatureNegotiator struct {
	steps []Step
	next  stream.FSMv0
}

// CloseStream handles properly closing a stream. It should be returned after a
// stream error has already been written to the stream.
type CloseStream struct {
}

// NewFeatureNegotiator creates a new FeatureNegotiator. It takes the next FSM
// to be executed and a slice of steps. The steps will be run in the order
// provided.
func NewFeatureNegotiator(next stream.FSMv0, steps []Step) FeatureNegotiator {
	return FeatureNegotiator{next: next, steps: steps}
}

// Next implements the stream.FSM interface.
func (fn FeatureNegotiator) Next(s stream.Streamv0) (stream.FSMv0, stream.Streamv0) {
	sfElem := element.StreamFeatures
	var exit, retry bool
	var restart bool = true

	for _, step := range fn.steps {
		fmt.Println(step)
		tries := 4
		if restart {
			err := s.Start()
			if err != nil {
				// TODO(skriptble): send a stream error?
				return CloseStream{}, s
			}
			var elems []element.Token
			for _, f := range step {
				elems = append(elems, f.TransformElement())
			}
			sfElem.Child = elems
			_, err = s.WriteElement(sfElem)
			if err != nil {
				// TODO(skriptble): send a stream error?
				return CloseStream{}, s
			}
			restart = false
		}
		firstround := true
		for firstround || retry || tries == 0 {
			log.Println("Here we go.")
			elem, err := s.Next()
			if err != nil {
				// TODO(skriptble): send a stream error?
				return CloseStream{}, s
			}
			log.Println(elem.Tag)
			f, ok := step[elem.Tag]
			if !ok {
				// TODO(skriptble): send a stream error
				tries--
				continue
			}
			s, restart, exit, retry = f.Negotiate(elem, s)
			log.Println("done")
			if exit {
				log.Println("oops")
				return CloseStream{}, s
			}
			firstround = false
		}
	}

	return fn.next, s
}

// Implements the stream.FSM interface.
func (cs CloseStream) Next(s stream.Streamv0) (stream.FSMv0, stream.Streamv0) {
	s.Write([]byte("</stream:stream>"))
	s.Close()
	return nil, s
}
