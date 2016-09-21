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

// ReceivingTCP is a stream transport for a receiving client, usually an XMPP
// server, that uses a TCP socket as described in RFC6120.
type ReceivingTCP struct {
	net.Conn
	*xml.Decoder
	// The stream is a weird XML element. Since we don't read the whole thing
	// we need to stash it's namespaces so its children can inherit them.
	streamNS map[string]string

	domain      string
	tlsRequired bool
	secure      bool
	conf        *tls.Config
	fgs         []stream.FeatureGenerator
}

// InitiatingTCP is a stream transport for an initiating client, usually an XMPP
// client or server, that uses a TCP socket as described in RFC6120.
type InitiatingTCP struct {
	net.Conn
	*xml.Decoder

	conf   *tls.Config
	secure bool
}

// NewReceivingTCP creates and returns a TCP stream.Transport
//
// tlsRequired will force tls upgrading of the stream before other features are
// negotiated.
//
// If conf is nil, the starttls feature will not be presented.
func NewReceivingTCP(c net.Conn, conf *tls.Config, tlsRequired bool, domain string, fgs []stream.FeatureGenerator) stream.Transport {
	dec := xml.NewDecoder(c)
	return &ReceivingTCP{
		Conn:        c,
		Decoder:     dec,
		conf:        conf,
		tlsRequired: tlsRequired,
		fgs:         fgs,
		domain:      domain,
	}
}

// WriteElement converts the element to bytes and writes to the underlying
// tcp connection. This method should generally be used for basic elements such
// as those used during SASL negotiation. WriteStanzas should be used when
// sending stanzas.
func (t *ReceivingTCP) WriteElement(el element.Element) error {
	var b []byte
	b = el.WriteBytes()
	_, err := t.Write(b)
	return err
}

// WriteStanzas converts the stanza to bytes and writes them to the underlying
// tcp connection. This method should be used whenever stanzas are being used
// instead of transforming the stanza to an element and using WriteElement.
func (t *ReceivingTCP) WriteStanza(st stanza.Stanza) error {
	el := st.TransformElement()
	return t.WriteElement(el)
}

func (it *InitiatingTCP) Next() (el element.Element, err error) {
	defer func() {
		if el.Tag == "features" && !it.secure {
			for _, ch := range el.ChildElements() {
				if ch.Tag == "starttls" {
					el, err = it.startTLS()
					break
				}
			}
		}
	}()
	return
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
func (t *ReceivingTCP) Next() (el element.Element, err error) {
	defer func() {
		if el.Tag == "starttls" && !t.secure {
			el, err = t.startTLS()
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

func (it *InitiatingTCP) startTLS() (el element.Element, err error) {
	// err = it.WriteElement(element.StartTLS)
	// if err != nil {
	// 	return
	// }
	// el, err = it.Next()
	// if err != nil || el.Tag != element.TLSProceed.Tag {
	// 	return
	// }
	// tlsConn = tls.Client(it.Conn, it.conf)
	// err = tlsConn.Handshake()
	// if err != nil {
	// 	return
	// }
	// conn := net.Conn(tlsConn)
	// it.Conn = conn
	// it.Decoder = xml.NewDecoder(conn)
	// el = element.Element{}
	// err = stream.ErrRequireRestart
	// it.secure = true
	// log.Println("Done upgrading connection")
	// return
	return
}

func (t *ReceivingTCP) startTLS() (el element.Element, err error) {
	var tlsConn *tls.Conn
	err = t.WriteElement(element.TLSProceed)
	if err != nil {
		return
	}
	tlsConn = tls.Server(t.Conn, t.conf)

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
func (t *ReceivingTCP) Start() (closed bool, err error) {
	// if t.mode == stream.Initiating {
	// 	if props.Header == (stream.Header{}) {
	// 		return props, stream.ErrHeaderNotSet
	// 	}
	// 	b := props.Header.WriteBytes()
	// 	_, err := t.Write(b)
	// 	return props, err
	// }

	// We're in recieving mode
	// if props.Domain == "" {
	// 	return props, stream.ErrDomainNotSet
	// }
	var el element.Element
	var h stream.Header

	el, err = t.Next()
	if err != nil {
		return false, err
	}

	h, err = stream.NewHeader(el)
	if err != nil {
		return false, err
	}

	h.ID = genStreamID()

	if h.To != t.domain {
		log.Printf("Host mismatch %v %v", h.To, t.domain)
		h.To, h.From = h.From, t.domain
		b := h.WriteBytes()
		t.Write(b)
		err = t.WriteElement(element.StreamError.HostUnknown)
		closed = true
		return closed, err
	}

	h.From, h.To = t.domain, h.From

	b := h.WriteBytes()
	_, err = t.Write(b)
	if err != nil {
		return false, err
	}

	// Stream features
	ftrs := element.StreamFeatures
	for _, fg := range t.fgs {
		el, ok := fg.GenerateFeature()
		if !ok {
			continue
		}
		ftrs = ftrs.AddChild(el)
	}
	if t.conf != nil && !t.secure {
		tlsFeature := element.StartTLS
		if t.tlsRequired {
			tlsFeature = tlsFeature.AddChild(element.Required)
			// Overwrite any other features
			ftrs.Child = []element.Token{}
		}
		ftrs = ftrs.AddChild(tlsFeature)
	}
	err = t.WriteElement(ftrs)
	return false, err
}

// createElement creates an element from the given xml.StartElement, populates
// its attributes and children and returns it.
func (t *ReceivingTCP) createElement(start xml.StartElement) (el element.Element, err error) {
	var children []element.Token

	composed := fmt.Sprintf("%s:%s", start.Name.Space, start.Name.Local)
	el = element.New(composed)
	if t.streamNS != nil {
		for k, v := range t.streamNS {
			el.Namespaces[k] = v
		}
	}
	for _, attr := range start.Attr {
		composed := fmt.Sprintf("%s:%s", attr.Name.Space, attr.Name.Local)
		el = el.AddAttr(composed, attr.Value)
	}
	el = el.ResetNamespace()
	// If this is a stream start return only this element.
	if el.Tag == "stream" && el.MatchNamespace(namespace.Stream) {
		t.streamNS = el.Namespaces
		return
	}

	children, err = t.childElements()
	el.Child = children
	return
}

// childElements retrieves child tokens. This method should be called after
// createElement.
func (t *ReceivingTCP) childElements() (children []element.Token, err error) {
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
