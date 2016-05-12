package transport

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"log"
	"net"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/stream"
)

// TCP is a stream transport that uses a TCP socket as described in RFC6120.
type TCP struct {
	net.Conn
	*xml.Decoder

	mode        stream.Mode
	tlsRequired bool
	conf        *tls.Config
	secure      bool
}

// NewTCP creates and returns a TCP stream.Transport
//
// tlsRequired will force tls upgrading of the stream before other features are
// negotiated.
//
// If conf is nil, the starttls feature will not be presented.
func NewTCP(c net.Conn, mode stream.Mode, conf *tls.Config, tlsRequired bool) stream.Transport {
	dec := xml.NewDecoder(c)
	return &TCP{Conn: c, Decoder: dec, mode: mode, conf: conf, tlsRequired: tlsRequired}
}

// WriteElement converts the element to bytes and writes to the underlying
// tcp connection. This method should generally be used for basic elements such
// as those used during SASL negotiation. WriteStanzas should be used when
// sending stanzas.
func (t *TCP) WriteElement(el element.Element) error {
	var b []byte
	b = el.WriteBytes()
	_, err := t.Write(b)
	return err
}

// WriteStanzas converts the stanza to bytes and writes them to the underlying
// tcp connection. This method should be used whenever stanzas are being used
// instead of transforming the stanza to an element and using WriteElement.
func (t *TCP) WriteStanza(st stanza.Stanza) error {
	el := st.TransformElement()
	return t.WriteElement(el)
}

// Next returns the next element from the stream. While most of the elements
// recieved from the stream are stanzas, this method is kept generic to allow
// handling stanzas and non-stanza elements such as those used during SASL
// neogitation.
//
// Since an element is the only valid thing an XML stream can return, this is
// the only method to read data from a transport.
//
// This transport hides the starttls upgrade feature so if a starttls element
// would have been returned, the connection is upgraded instead.
func (t *TCP) Next() (el element.Element, err error) {
	defer func() {
		if el.Tag == "starttls" && !t.secure {
			el, err = t.startTLS()
		}
	}()
	defer func() {
		if el.Tag == "features" && !t.secure {
			for _, ch := range el.ChildElements() {
				if ch.Tag == "starttls" {
					el, err = t.startTLS()
					break
				}
			}
		}
	}()
	var token xml.Token
	for {
		token, err = t.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			return t.createElement(elem)
		case xml.EndElement:
			err = stream.ErrStreamClosed
			return
		}
	}
}

func (t *TCP) startTLS() (el element.Element, err error) {
	var tlsConn *tls.Conn
	if t.mode == stream.Initiating {
		err = t.WriteElement(element.StartTLS)
		if err != nil {
			return
		}
		el, err = t.Next()
		if err != nil || el.Tag != element.TLSProceed.Tag {
			return
		}
		tlsConn = tls.Client(t.Conn, t.conf)
	} else {
		err = t.WriteElement(element.TLSProceed)
		if err != nil {
			return
		}
		tlsConn = tls.Server(t.Conn, t.conf)
	}

	err = tlsConn.Handshake()
	if err != nil {
		return
	}
	conn := net.Conn(tlsConn)
	t.Conn = conn
	t.Decoder = xml.NewDecoder(conn)
	el = element.Element{}
	err = stream.ErrRequireRestart
	t.secure = true
	log.Println("Done upgrading connection")
	return
}

// Start starts or restarts the stream.
//
// In recieving mode, the transport will wait to recieve a stream header
// from the initiating entity, then sends its own header and the stream
// features. This transport will add the starttls feature under certain
// conditions.
func (t *TCP) Start(props stream.Properties) (stream.Properties, error) {
	if t.mode == stream.Initiating {
		if props.Header == (stream.Header{}) {
			return props, stream.ErrHeaderNotSet
		}
		b := props.Header.WriteBytes()
		_, err := t.Write(b)
		return props, err
	}

	// We're in recieving mode
	if props.Domain == "" {
		return props, stream.ErrDomainNotSet
	}
	var el element.Element
	var h stream.Header
	var err error

	el, err = t.Next()
	if err != nil {
		return props, err
	}

	h, err = stream.NewHeader(el)
	if err != nil {
		return props, err
	}

	h.ID = genStreamID()

	if h.To != props.Domain {
		h.To, h.From = h.From, props.Domain
		b := h.WriteBytes()
		t.Write(b)
		err = t.WriteElement(element.StreamError.HostUnknown)
		props.Status = stream.Closed
		return props, err
	}

	h.From, h.To = props.Domain, h.From
	if props.To != "" {
		h.To = props.To
	}

	props.Header = h

	b := props.Header.WriteBytes()
	_, err = t.Write(b)
	if err != nil {
		return props, err
	}

	ftrs := element.StreamFeatures
	for _, f := range props.Features {
		ftrs = ftrs.AddChild(f)
	}
	// Stream features
	if t.conf != nil && !t.secure {
		tlsFeature := element.StartTLS
		if t.tlsRequired {
			tlsFeature = tlsFeature.AddChild(element.Required)
		}
		// Overwrite any other features
		ftrs.Child = []element.Token{tlsFeature}
	}
	err = t.WriteElement(ftrs)
	return props, err
}

// createElement creates an element from the given xml.StartElement, populates
// its attributes and children and returns it.
func (t *TCP) createElement(start xml.StartElement) (el element.Element, err error) {
	var children []element.Token

	el = element.Element{
		Space: start.Name.Space,
		Tag:   start.Name.Local,
	}
	for _, attr := range start.Attr {
		el.Attr = append(
			el.Attr,
			element.Attr{
				Space: attr.Name.Space,
				Key:   attr.Name.Local,
				Value: attr.Value,
			},
		)
	}
	// If this is a stream start return only this element.
	if el.Tag == "stream" && el.Space == namespace.Stream {
		return
	}

	children, err = t.childElements()
	el.Child = children
	return
}

// childElements retrieves child tokens. This method should be called after
// createElement.
func (t *TCP) childElements() (children []element.Token, err error) {
	var token xml.Token
	var el element.Element
	for {
		token, err = t.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			el, err = t.createElement(elem)
			if err != nil {
				return
			}
			children = append(children, el)
		case xml.EndElement:
			return
		case xml.CharData:
			data := string(elem)
			children = append(children, element.CharData{Data: data})
		}
	}
}

// genStreamID creates a new stream ID based on a uuid.
func genStreamID() string {
	id := make([]byte, 16)
	rand.Read(id)

	id[8] = (id[8] | 0x80) & 0xBF
	id[6] = (id[6] | 0x40) & 0x4F

	return fmt.Sprintf("ni%xne", id)
}
