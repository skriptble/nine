package stream

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

// ErrRequireRestart is the error returned when the underlying transport
// has been upgraded and the stream needs to be restarted.
var ErrRequireRestart = errors.New("Transport upgrade. Restart stream.")

// Trace is the trace logger for the stream package. Outputs useful
// tracing information.
var Trace *log.Logger = log.New(ioutil.Discard, "[TRACE] [stream] ", log.LstdFlags|log.Lshortfile)

// Debug is the debug logger for the stream package. Outputs useful
// debugging information.
var Debug *log.Logger = log.New(ioutil.Discard, "[DEBUG] [stream] ", log.LstdFlags|log.Lshortfile)

type Status int

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

const (
	Receiving Mode = iota
	Initiating
)

type Properties struct {
	Header
	Status

	// The XMPP domain of this server.
	Domain   string
	Features []element.Element
}

func NewProperties() Properties {
	return Properties{}
}

type FeatureHandler interface {
	HandleFeature(Properties) Properties
}

type Transport interface {
	io.Closer

	WriteElement(el element.Element) error
	WriteStanza(st stanza.Stanza) error
	Next() (el element.Element, err error)
	Start(Properties) (Properties, error)
}

type UpgradeableTransport interface {
	Transport

	// Upgrade upgrades the unlderying transport. Returns true if the transport
	// was upgraded TLS.
	Upgrade() (Transport, bool)
}

type Stream struct {
	Properties

	h         ElementHandler
	t         Transport
	fhs       []FeatureHandler
	eHandlers []ElementMux
	mHandlers []MessageMux
	pHandlers []PresenceMux
	iHandlers []IQMux

	mode   Mode
	strict bool
}

// New creates a new stream using the underlying trasport. The properties
// make up the initial set of properties for the stream.
//
// Mode allows a stream to be used as either the initiating entity or the
// receiving entity.
//
// Strict indicates how strict to RFC-6120 the stream operates. For example, if
// strict is set to true then a stream error will for a close of the stream.
func New(t Transport, h ElementHandler, mode Mode) Stream {
	return Stream{t: t, h: h, mode: mode}
}

func (s Stream) SetProperties(p Properties) Stream {
	s.Properties = p
	return s
}

// AddElementHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddElementHandlers(hdlrs ...ElementMux) Stream {
	s.eHandlers = append(s.eHandlers, hdlrs...)
	return s
}

// AddMessageHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddMessageHandlers(hdlrs ...MessageMux) Stream {
	s.mHandlers = append(s.mHandlers, hdlrs...)
	return s
}

// AddPresenceHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddPresenceHandlers(hdlrs ...PresenceMux) Stream {
	s.pHandlers = append(s.pHandlers, hdlrs...)
	return s
}

// AddIQHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddIQHandlers(hdlrs ...IQMux) Stream {
	s.iHandlers = append(s.iHandlers, hdlrs...)
	return s
}

// AddFeatureHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddFeatureHandlers(hdlrs ...FeatureHandler) Stream {
	s.fhs = append(s.fhs, hdlrs...)
	return s
}

func syntaxError(err error) bool {
	_, ok := err.(*xml.SyntaxError)
	return ok
}

func networkError(err error) bool {
	_, ok := err.(net.Error)
	return ok || err == io.EOF
}

// TODO(skriptble): How should errors from running the stream be handled?
func (s Stream) Run() {
	var err error
	// Start the stream
	Trace.Println("Running stream.")
	s.Properties.Status = s.Properties.Status | Restart

	// Start recieving elements
	for {
		// Restart stream as necessary
		if s.Properties.Status&Restart != 0 {
			Trace.Println("(Re)starting stream.")
			s.Properties.Features = []element.Element{}
			for _, fh := range s.fhs {
				s.Properties = fh.HandleFeature(s.Properties)
			}
			s.Properties, err = s.t.Start(s.Properties)
			if err != nil {
				Debug.Printf("Error while restarting stream: %s", err)
			}
			// If the restart bit is still on
			// TODO: Should this always be handled by the transport?
			if s.Properties.Status&Restart != 0 {
				s.Properties.Status = s.Properties.Status ^ Restart
			}
		}

		el, err := s.t.Next()
		if err != nil {
			Trace.Printf("Error recieved: %s", err)
			switch {
			case err == ErrRequireRestart:
				s.Properties.Status = s.Properties.Status | Restart
				Trace.Println("Restart setup")
				continue
			case syntaxError(err):
				Debug.Println("XML Syntax Error", err)
				err = s.t.WriteElement(element.StreamErrBadFormat)
				if s.strict {
					s.t.Close()
					return
				}
			case networkError(err):
				Debug.Printf("Network error. Stopping. err: %s", err)
				return
			case err == ErrStreamClosed:
				Trace.Println("Stream close recieved. Closing stream.")
				s.t.Close()
				return
			}
		} else {

		}

		var elems []element.Element
		Trace.Printf("Element: %s", el)
		elems, s.Properties = s.h.HandleElement(el, s.Properties)
		for _, elem := range elems {
			Trace.Printf("Writing element: %s", elem)
			s.t.WriteElement(elem)
		}

		// if iq, err := stanza.TransformIQ(el); err == nil {
		// 	Trace.Printf("Element is IQ: %s", iq)
		// 	Trace.Println("Running IQ Handlers")
		// 	for _, h := range s.iHandlers {
		// 		if !h.Match(iq) {
		// 			continue
		// 		}
		// 		Trace.Println("Match Found")
		// 		var sts []stanza.Stanza
		// 		sts, s.Properties = h.FSM.HandleIQ(iq, s.Properties)
		// 		for _, st := range sts {
		// 			Trace.Printf("Writing stanza: %s", st)
		// 			s.t.WriteStanza(st)
		// 		}
		// 		break
		// 	}
		// }

		// if presence, err := stanza.TransformPresence(el); err == nil {
		// 	Trace.Printf("Element is Presence: %s", presence)
		// 	Trace.Println("Running Presence Handlers")
		// 	for _, h := range s.pHandlers {
		// 		if !h.Match(presence) {
		// 			continue
		// 		}
		// 		break
		// 	}
		// }

		// if message, err := stanza.TransformMessage(el); err == nil {
		// 	Trace.Printf("Element is Message: %s", message)
		// 	Trace.Println("Running Message Handlers")
		// 	for _, h := range s.mHandlers {
		// 		if !h.Match(message) {
		// 			continue
		// 		}
		// 	}
		// }

	}
}
