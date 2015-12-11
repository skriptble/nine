package features

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
	"github.com/skriptble/nine/stream/transport"
)

// StartTLS is used for the server side of the TLS stream feature negotiation.
type StartTLS struct {
	required bool
	config   *tls.Config
}

// NewStartTLS creates a Feature that can be used to upgrade a Stream to TLS.
func NewStartTLS(config *tls.Config, required bool) Feature {
	return StartTLS{
		required: required,
		config:   config,
	}
}

// Negotiate implements the feature.Feature interface.
func (stls StartTLS) Negotiate(elem element.Element, s stream.Streamv0) (ns stream.Streamv0, restart, closeStream, retry bool) {
	if elem.Tag != "starttls" {
		// TODO(skriptble): Send a stream error
		return s, false, true, false
	}
	tcpTransport, ok := s.Transportv0.(transport.TCPv0)
	if !ok {
		// TODO(skriptble): Send a stream error (maybe?)
		return s, false, true, false
	}
	_, err := s.WriteElement(element.TLSProceed)
	if err != nil {
		// TODO(skriptble): Send a stream error
		return s, false, true, false
	}
	log.Println("Making TLS magic happen!")
	tlsConn := tls.Server(tcpTransport.Conn, stls.config)
	tlsConn.Handshake()
	log.Println("Done shaking hands!")

	conn := net.Conn(tlsConn)
	s = stream.Newv0(transport.NewTCPv0(conn), stream.Receiving)
	// s.Transport = tcpTransport
	// s.Header = stream.Header{}

	return s, true, false, false
}

// TransformElement implements the feature.Feature interface.
func (stls StartTLS) TransformElement() element.Element {
	elem := element.StartTLS
	if stls.required {
		elem.Child = append(elem.Child, element.Element{Tag: "required"})
	}
	return elem
}
