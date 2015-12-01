package stream

import (
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/skriptble/nine/element"
)

// Mode determines the mode of the stream.
//
// Currently this is either Initiating or Receiving for the stream initiating
// entity or receiving entity, respectively.
type Mode int

const (
	Initiating Mode = iota
	Receiving
)

// ErrStreamClosed is the error returned when the stream has been closed.
var ErrStreamClosed = errors.New("Stream Closed")

// ErrHeaderNotSet is the error returned when start has been called on a stream
// in initiating mode and the header has not yet been set.
var ErrHeaderNotSet = errors.New("Stream Header has not been set")

// Stream represents an XML Stream
type Stream struct {
	Header

	*xml.Decoder
	rw   io.ReadWriter
	mode Mode
}

// New creates a new Stream from the given io.ReadWriter.
//
// The mode parameter determines if this the initiating or recieving entity.
func New(rw io.ReadWriter, mode Mode) Stream {
	dec := xml.NewDecoder(rw)
	return Stream{Decoder: dec, rw: rw, mode: mode}
}

// Start writes the stream header to the stream.
//
// If the stream is in initiating mode, the Header must be set before this
// method is called.
//
// If the server is in receiving mode and the Header has not been set this
// method will read a token from the stream, if the token is a stream header
// it will generate a new stream id, flip the to and from fields, and set the
// Header field.
//
// If the conditions are met for either mode, the stream header is sent and any
// errors are returned.
//
// TODO(skriptble): Should there be a default from for the recieving mode?
// Or should an error be returned if there is no from after the flip?
func (s Stream) Start() (err error) {
	if s.mode == Initiating {
		if s.Header == (Header{}) {
			return ErrHeaderNotSet
		}
		_, err = s.writeHeader()
		return
	}

	if s.Header == (Header{}) {
		var elem element.Element
		var hdr Header
		var id string

		elem, err = s.Next()
		if err != nil {
			return
		}

		hdr, err = NewHeader(elem)
		if err != nil {
			return
		}

		id, err = genStreamID()
		if err != nil {
			return
		}

		hdr.ID = id
		hdr.To, hdr.From = hdr.From, hdr.To
		s.Header = hdr
	}

	_, err = s.writeHeader()
	return
}

// TODO(skriptble): Does this need to return the bytes sent?
func (s Stream) writeHeader() (n int, err error) {
	b := s.Header.WriteBytes()
	return s.Write(b)
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
			err = ErrStreamClosed
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

func genStreamID() (string, error) {
	id := make([]byte, 16)
	_, err := rand.Read(id)
	if err != nil {
		return "", err
	}

	id[8] = (id[8] | 0x80) & 0xBF
	id[6] = (id[6] | 0x40) & 0x4F

	return fmt.Sprintf("%x", id), nil
}
