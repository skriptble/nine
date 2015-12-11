package stream

import (
	"encoding/xml"
	"errors"
	"io"
	"log"
	"net"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

// ErrRequireRestart is the error returned when the underlying transport
// has been upgraded and the stream needs to be restarted.
var ErrRequireRestart = errors.New("Transport upgrade. Restart stream.")

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

	mode Mode
}

func New(t Transport, p Properties, mode Mode, eh []ElementHandler, fh []FeatureHandler) Stream {
	return Stream{t: t, Properties: p, mode: mode, eHandlers: eh, fhs: fh}
}

// TODO(skriptble): How should erros from running the stream be handled?
func (s Stream) Run() {
	// Start the stream
	var props Properties = s.Properties
	var err error
	for _, fh := range s.fhs {
		props = fh.HandleFeature(props)
	}
	s.Properties = props
	s.Properties, err = s.t.Start(s.Properties)
	if err != nil {
		log.Printf("Error while starting stream: %s", err)
	}
	// Start recieving elements
	for {
		el, err := s.t.Next()
		if err != nil {
			if err == ErrRequireRestart {
				s.Properties.Status = s.Properties.Status | Restart
				log.Println("Restart setup")
			}
			if _, ok := err.(*xml.SyntaxError); ok {
				log.Println("XML Syntax Error", err)
				err = s.t.WriteElement(element.StreamErrBadFormat)
			}
			if _, ok := err.(net.Error); ok || err == io.EOF {
				break
			}
			if err == ErrStreamClosed {
				s.t.Close()
				break
			}
		}

		if el.Tag == "starttls" {
			ut, ok := s.t.(UpgradeableTransport)
			if !ok {
				continue
			}
			t, success := ut.Upgrade()
			s.t = t
			if success {
				s.Properties.Status = s.Properties.Status | Secure
			}
			s.Properties.Features = []element.Element{}
			for _, fh := range s.fhs {
				s.Properties = fh.HandleFeature(s.Properties)
			}
			s.Properties, err = s.t.Start(s.Properties)
			if err != nil {
				log.Printf("Error while starting stream: %s", err)
			}
		}

		if iq, err := stanza.TransformIQ(el); err == nil {
			for _, h := range s.iHandlers {
				if !h.Match(iq) {
					continue
				}
			}
		}

		log.Println("HANDLER TIME")
		for i, h := range s.eHandlers {
			if !h.Match(el) {
				continue
			}
			var elems []element.Element
			elems, s.eHandlers[i].FSM, s.Properties = h.FSM.HandleElement(el, s.Properties)
			for _, elem := range elems {
				s.t.WriteElement(elem)
			}
		}

		// Restart stream as necessary
		if s.Properties.Status&Restart != 0 {
			s.Properties.Features = []element.Element{}
			for _, fh := range s.fhs {
				s.Properties = fh.HandleFeature(s.Properties)
			}
			s.Properties, err = s.t.Start(s.Properties)
			if err != nil {
				log.Printf("Error while starting stream: %s", err)
			}
		}
		// Handle elements
	}
}
