package transport

import (
	"encoding/xml"
	"net"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

// TCPTransport is the transport used for a regular TCP XMPP connection.
type TCPv0 struct {
	net.Conn

	*xml.Decoder
}

func NewTCPv0(c net.Conn) stream.Transportv0 {
	dec := xml.NewDecoder(c)
	return TCPv0{Conn: c, Decoder: dec}
}

func (tcpt TCPv0) WriteHeader(h stream.Header) (err error) {
	b := h.WriteBytes()
	_, err = tcpt.Write(b)
	return err
}

// WriteElement writes an element to the stream.
func (tcpt TCPv0) WriteElement(el element.Element) (n int, err error) {
	var b []byte
	b = el.WriteBytes()
	return tcpt.Write(b)
}

// Next returns the next element from the stream. Since an element is the only
// valid thing an XML stream can return, this is the only method to read data
// from a stream.
func (tcpt TCPv0) Next() (el element.Element, err error) {
	var token xml.Token
	for {
		token, err = tcpt.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			return tcpt.createElement(elem)
		case xml.EndElement:
			err = stream.ErrStreamClosed
			return
		}
	}
}

// createElement creates an element from the given xml.StartElement, populates
// its attributes and children and returns it.
func (tcpt TCPv0) createElement(start xml.StartElement) (el element.Element, err error) {
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
	if el.Tag == "stream" {
		return
	}

	children, err = tcpt.childElements()
	el.Child = children
	return
}

// childElements retrieves child tokens. This method should be called after
// createElement.
func (tcpt TCPv0) childElements() (children []element.Token, err error) {
	var token xml.Token
	var el element.Element
	for {
		token, err = tcpt.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			el, err = tcpt.createElement(elem)
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
