package stream

import (
	"encoding/xml"
	"errors"
	"io"

	"github.com/skriptble/nine/element"
)

// StreamClosed is the error returned when the stream has been closed.
var StreamClosed = errors.New("Stream Closed")

// Stream represents an XML Stream
type Stream struct {
	*xml.Decoder
	rw io.ReadWriter
}

// New creates a new Stream from the given io.ReadWriter
func New(rw io.ReadWriter) Stream {
	dec := xml.NewDecoder(rw)
	return Stream{Decoder: dec, rw: rw}
}

// Write allows the writting of arbitrary bytes to the stream.
func (s Stream) Write(p []byte) (n int, err error) {
	return s.rw.Write(p)
}

// WriteElement writes an element to the stream.
func (s Stream) WriteElement(el element.Element) (n int, err error) {
	var b []byte
	b = el.WriteBytes()
	return s.rw.Write(b)
}

// Next returns the next element from the stream. Since an element is the only
// valid thing and XML stream can return, this is the only method to read data
// from a stream.
func (s Stream) Next() (el element.Element, err error) {
	var token xml.Token
	for {
		token, err = s.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			return s.createElement(elem)
		case xml.EndElement:
			err = StreamClosed
			return
		}
	}
}

// createElement creates an element from the given xml.StartElement, populates
// its attributes and children and returns it.
func (s Stream) createElement(start xml.StartElement) (el element.Element, err error) {
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

	children, err = s.childElements()
	el.Child = children
	return
}

// childElements retrieves child tokens. This method should be called after
// createElement.
func (s Stream) childElements() (children []element.Token, err error) {
	var token xml.Token
	var el element.Element
	for {
		token, err = s.Token()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			el, err = s.createElement(elem)
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
