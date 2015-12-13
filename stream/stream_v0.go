package stream

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/skriptble/nine/element"
)

// ErrStreamClosed is the error returned when the stream has been closed.
var ErrStreamClosed = errors.New("Stream Closed")

// ErrHeaderNotSet is the error returned when start has been called on a stream
// in initiating mode and the header has not yet been set.
var ErrHeaderNotSet = errors.New("Stream Header has not been set")

// ErrNilTransport is the error returned when the Transport for a stream has
// not been set.
var ErrNilTransport = errors.New("Stream Transport is not set")

// Stream represents an XML Stream
type Streamv0 struct {
	Transportv0

	Header

	mode Mode
}

// Transport represents the underlying transport for an XML Stream
type Transportv0 interface {
	io.ReadWriteCloser

	WriteElement(el element.Element) (n int, err error)
	WriteHeader(Header) error
	Next() (el element.Element, err error)
}

// New creates a new Stream from the given io.ReadWriter.
//
// The mode parameter determines if this the initiating or recieving entity.
func Newv0(transport Transportv0, mode Mode) Streamv0 {
	return Streamv0{Transportv0: transport, mode: mode}
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
func (s Streamv0) Start() (err error) {
	if s.mode == Initiating {
		if s.Header == (Header{}) {
			return ErrHeaderNotSet
		}
		err = s.WriteHeader(s.Header)
		return
	}

	if s.Header == (Header{}) {
		var elem element.Element
		var hdr Header
		var id string

		log.Println("Starting the stream...")
		elem, err = s.Next()
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Stream Header :%s", elem.WriteBytes())
		log.Printf("Stream (Go): %+v", elem)

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

	err = s.WriteHeader(s.Header)
	log.Println("Stream started.")
	return
}

// Next provides a default implementation for the Trasport, handles a nil
// Transport to avoid panics.
func (s Streamv0) Next() (el element.Element, err error) {
	if s.Transportv0 == nil {
		err = ErrNilTransport
		return
	}
	return s.Transportv0.Next()
}

// WriteElement provides a default implementation for the Trasport, handles a
// nil Transport to avoid panics.
func (s Streamv0) WriteElement(el element.Element) (n int, err error) {
	if s.Transportv0 == nil {
		err = ErrNilTransport
		return
	}
	return s.Transportv0.WriteElement(el)
}

// Read provides a default implementation for the Trasport, handles a nil
// Transport to avoid panics.
func (s Streamv0) Read(p []byte) (n int, err error) {
	if s.Transportv0 == nil {
		err = ErrNilTransport
		return
	}
	return s.Transportv0.Read(p)
}

// Write provides a default implementation for the Trasport, handles a nil
// Transport to avoid panics.
func (s Streamv0) Write(p []byte) (n int, err error) {
	if s.Transportv0 == nil {
		err = ErrNilTransport
		return
	}
	return s.Transportv0.Write(p)
}

// WriteHeader provides a default implementation for the Trasport, handles a nil
// Transport to avoid panics.
func (s Streamv0) WriteHeader(h Header) (err error) {
	if s.Transportv0 == nil {
		err = ErrNilTransport
		return
	}
	return s.Transportv0.WriteHeader(h)
}

// Close provides a default implementation for the Trasport, handles a nil
// Transport to avoid panics.
func (s Streamv0) Close() error {
	if s.Transportv0 == nil {
		return ErrNilTransport
	}
	return s.Transportv0.Close()
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
