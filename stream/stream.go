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
	Closed Status = iota
	Open   Status = 1 << iota
	Restart
	Secure
	Auth
	Bind
)

type Properties struct {
	Header
	Status

	Features []element.Element
}

func NewProperties() Properties {
	return Properties{Status: Open}
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

	t         Transport
	fhs       []FeatureHandler
	eHandlers []ElementHandler
	mHandlers []MessageHandler
	pHandlers []PresenceHandler
	iHandlers []IQHandler

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
func New(t Transport, p Properties, mode Mode, strict bool) Stream {
	return Stream{t: t, Properties: p, mode: mode, strict: strict}
}

// AddElementHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddElementHandlers(hdlrs ...ElementHandler) Stream {
	s.eHandlers = append(s.eHandlers, hdlrs...)
	return s
}

// AddMessageHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddMessageHandlers(hdlrs ...MessageHandler) Stream {
	s.mHandlers = append(s.mHandlers, hdlrs...)
	return s
}

// AddPresenceHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddPresenceHandlers(hdlrs ...PresenceHandler) Stream {
	s.pHandlers = append(s.pHandlers, hdlrs...)
	return s
}

// AddIQHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddIQHandlers(hdlrs ...IQHandler) Stream {
	s.iHandlers = append(s.iHandlers, hdlrs...)
	return s
}

// AddFeatureHandlers appends the given handlers to the end of the handlers
// for the stream.
func (s Stream) AddFeatureHandlers(hdlrs ...FeatureHandler) Stream {
	s.fhs = append(s.fhs, hdlrs...)
	return s
}

// TODO(skriptble): How should errors from running the stream be handled?
func (s Stream) Run() {
	// Start the stream
	Trace.Println("Running stream.")
	var props Properties = s.Properties
	var err error
	for _, fh := range s.fhs {
		props = fh.HandleFeature(props)
	}
	s.Properties = props

	Trace.Println("Starting stream.")
	s.Properties, err = s.t.Start(s.Properties)
	if err != nil {
		Debug.Printf("Error while starting stream: %s", err)
	}

	// Start recieving elements
	for {
		el, err := s.t.Next()
		if err != nil {
			Trace.Printf("Error recieved: %s", err)
			if err == ErrRequireRestart {
				s.Properties.Status = s.Properties.Status | Restart
				Trace.Println("Restart setup")
			}
			if _, ok := err.(*xml.SyntaxError); ok {
				Debug.Println("XML Syntax Error", err)
				err = s.t.WriteElement(element.StreamErrBadFormat)
				if s.strict {
					s.t.Close()
					break
				}
			}
			if _, ok := err.(net.Error); ok || err == io.EOF {
				Debug.Printf("Network error. Stopping. err: %s", err)
				break
			}
			if err == ErrStreamClosed {
				Trace.Println("Stream close recieved. Closing stream.")
				s.t.Close()
				break
			}
		}

		if iq, err := stanza.TransformIQ(el); err == nil {
			Trace.Printf("Element is IQ: %s", iq)
			Trace.Println("Running IQ Handlers")
			for _, h := range s.iHandlers {
				if !h.Match(iq) {
					continue
				}
				Trace.Println("Match Found")
				var sts []stanza.Stanza
				sts, s.Properties = h.FSM.HandleIQ(iq, s.Properties)
				for _, st := range sts {
					Trace.Printf("Writing stanza: %s", st)
					s.t.WriteStanza(st)
				}
				break
			}
		}

		if presence, err := stanza.TransformPresence(el); err == nil {
			Trace.Printf("Element is Presence: %s", presence)
			Trace.Println("Running Presence Handlers")
			for _, h := range s.pHandlers {
				if !h.Match(presence) {
					continue
				}
				break
			}
		}

		if message, err := stanza.TransformMessage(el); err == nil {
			Trace.Printf("Element is Message: %s", message)
			Trace.Println("Running Message Handlers")
			for _, h := range s.mHandlers {
				if !h.Match(message) {
					continue
				}
			}
		}

		// Handle elements
		Trace.Println("Running Element Handlers")
		for _, h := range s.eHandlers {
			if !h.Match(el) {
				continue
			}
			var elems []element.Element
			elems, s.Properties = h.FSM.HandleElement(el, s.Properties)
			for _, elem := range elems {
				Trace.Printf("Writing element: %s", elem)
				s.t.WriteElement(elem)
			}
		}

		// Restart stream as necessary
		if s.Properties.Status&Restart != 0 {
			Trace.Println("Restarting stream.")
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
	}
}
