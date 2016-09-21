package stream

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"runtime/debug"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

// ErrStreamClosed is the error returned when the stream has been closed.
var ErrStreamClosed = errors.New("Stream Closed")

// ErrHeaderNotSet is the error returned when start has been called on a stream
// in initiating mode and the header has not yet been set.
var ErrHeaderNotSet = errors.New("Stream Header has not been set")

// ErrDomainNotSet is the error returned when start has been called on a
// stream in initiating mode and the header has not yet been set.
var ErrDomainNotSet = errors.New("Stream Domain has not been set")

// ErrNilTransport is the error returned when the Transport for a stream has
// not been set.
var ErrNilTransport = errors.New("Stream Transport is not set")

// ErrRequireRestart is the error returned when the underlying transport
// has been upgraded and the stream needs to be restarted.
var ErrRequireRestart = errors.New("Transport upgrade. Restart stream.")

// Trace is the trace logger for the stream package. Outputs useful
// tracing information.
var Trace = log.New(ioutil.Discard, "[TRACE] [stream] ", log.LstdFlags|log.Lshortfile)

// Debug is the debug logger for the stream package. Outputs useful
// debugging information.
var Debug = log.New(ioutil.Discard, "[DEBUG] [stream] ", log.LstdFlags|log.Lshortfile)

// Status represents the states of a stream. It is used to determine if the
// stream is open, closed, needs to be restarted, is authenticated, or has been
// bound.
type Status int

// The statuses of a stream. They are implementated as bits so each one can be
// set indepedently. If Closed is set, all other bits are usually ignored.
// The zero value of a status is open, which means it requires no
// initialization.
const (
	Open   Status = iota
	Closed Status = 1 << iota
	Restart
	Secure
	Auth
	Bind
)

// Mode determines the mode of the stream.
//
// Currently this is either Initiating or Receiving for the stream initiating
// entity or receiving entity, respectively.
type Mode int

// There are two modes for a stream: Receiving and Initating. These are mostly
// used at the transport layer to determine how things like restarting and
// starting a stream are handled.
const (
	Receiving Mode = iota
	Initiating
)

// Properties is a collection of properties of a stream. It is passed to
// Handlers and Transports which can modify and return the properties. This is
// mainly used as a statemachine to help determine what state a stream is in
// and affect how Elements are handled. It is also used in the stream starting
// and restarting process for things like the currently available features of a
// stream.
type Properties struct {
	Header
	Status

	// The XMPP domain of this server.
	Domain   string
	Features []element.Element
}

// NewProperties initializes and returns a Properties object.
func NewProperties() Properties {
	return Properties{}
}

// FeatureGenerator is the interface implemented by a type that can create and
// return a feature element. These features are then sent to an initiating
// entity during stream negotiation. If no feature element was generated, ok
// should be false.
type FeatureGenerator interface {
	GenerateFeature() (el element.Element, ok bool)
}

// Transport is the interface implemented by types that can handle low level
// stream features such as reading an element, writing an element, and starting
// a stream.
type Transport interface {
	io.WriteCloser

	WriteElement(el element.Element) error
	WriteStanza(st stanza.Stanza) error
	Next() (el element.Element, err error)
	Start() (close bool, err error)
}

// Stream represents an RFC6120 stream.
//
// It is written in a functional style: most of the methods return a new stream
// object instead of modifying the one passed in.
type Stream struct {
	h       ElementHandlerV2
	t       Transport
	restart bool
	close   bool

	mode Mode
}

// New creates a new stream using the underlying trasport. The properties
// make up the initial set of properties for the stream.
//
// Mode allows a stream to be used as either the initiating entity or the
// receiving entity.
func New(t Transport, h ElementHandlerV2, mode Mode) Stream {
	return Stream{t: t, h: h, mode: mode}
}

func syntaxError(err error) bool {
	_, ok := err.(*xml.SyntaxError)
	return ok
}

func networkError(err error) bool {
	_, ok := err.(net.Error)
	return ok || err == io.EOF
}

// Run is the main execution thread of the stream. It handles starting the
// stream and then retrieving elements. The elements retrieved are passed to
// the stream's element handler.
//
// Run will return when an error has occured or the stream has closed. This
// should only be called once, although calling it more than once won't cause
// a panic. The functionality of the stream if Run is called more than once is
// undefined.
func (s Stream) Run() {
	defer func() {
		if r := recover(); r != nil {
			// Something panicked so our state is probably bad, cleanly shut
			// down and return.
			Debug.Println("panic occurred during Run. Cleaning up")
			Debug.Printf("%s\n", debug.Stack())
			s.t.WriteElement(element.StreamError.InternalServerError)
			s.t.Close()
			return
		}
	}()
	var err error
	// Start the stream
	Trace.Println("Running stream.")
	Trace.Println("Starting stream.")
	s.close, err = s.t.Start()
	if err != nil {
		if syntaxError(err) {
			Debug.Println("XML Syntax Error", err)
			s.t.WriteElement(element.StreamErrBadFormat)
			s.t.Close()
			return
		}
		Debug.Printf("Error while starting stream: %s", err)
	}

	var elems []element.Element
	// Start recieving elements
	for {
		// Restart stream as necessary
		if s.restart {
			Trace.Println("Restarting stream.")
			// TODO(skriptble): This should only be called for streams in receiving mode.
			s.close, err = s.t.Start()
			if err != nil {
				if syntaxError(err) {
					Debug.Println("XML Syntax Error", err)
					s.t.WriteElement(element.StreamErrBadFormat)
					s.t.Close()
					return
				}
				Debug.Printf("Error while restarting stream: %s", err)
			}
			s.restart = false
		}

		el, err := s.t.Next()
		if err != nil {
			Trace.Printf("Error received: %s", err)
			switch {
			case err == ErrRequireRestart:
				s.restart = true
				Trace.Println("Restart setup")
				continue
			case syntaxError(err):
				Debug.Println("XML Syntax Error", err)
				err = s.t.WriteElement(element.StreamErrBadFormat)
				s.t.Close()
				return
			case networkError(err):
				Debug.Printf("Network error. Stopping. err: %s", err)
				return
			case err == ErrStreamClosed:
				Trace.Println("Stream close received. Closing stream.")
				s.t.Write([]byte("</stream:stream>"))
				s.t.Close()
				return
				// TODO: Add a default case. This should probably close the stream.
			}
		}

		Trace.Printf("Element: %s", el)
		elems, s.restart, s.close = s.h.HandleElement(el)
		for _, elem := range elems {
			Trace.Printf("Writing element: %s", elem)
			s.t.WriteElement(elem)
		}
		if s.close {
			Trace.Println("Handler closed stream. Closing stream.")
			s.t.Write([]byte("</stream:stream>"))
			s.t.Close()
			return
		}
	}
}

func (s Stream) WriteElement(el element.Element) error {
	return s.t.WriteElement(el)
}

func (s Stream) WriteStanza(st stanza.Stanza) error {
	return s.t.WriteStanza(st)
}

func (s Stream) Close() error {
	return s.t.Close()
}
