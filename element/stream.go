package element

import (
	"bytes"
	"fmt"

	"github.com/skriptble/nine/namespace"
)

// Stream represents an XML stream element.
type StreamHeader struct {
	To, From  string
	ID        string
	Lang      string
	Version   string
	Namespace string
}

// NewStream attempts to transform the Element into a Stream. Returns an error
// if the element is not a stream element.
func NewStreamHeader(el Element) (strm StreamHeader, err error) {
	if el.Space != "stream" && el.Tag != "stream" {
		err = fmt.Errorf("Element is not <stream:stream> it is a <%s:%s>", el.Space, el.Tag)
		return
	}

	for _, attr := range el.Attr {
		switch attr.Key {
		case "to":
			strm.To = attr.Value
		case "from":
			strm.From = attr.Value
		case "lang":
			if attr.Space == "xml" {
				strm.Lang = attr.Value
			}
		case "version":
			strm.Version = attr.Value
		case "xmlns":
			strm.Namespace = attr.Value
		}
	}
	return
}

// WriteBytes writes the header to bytes.
//
// This is done instead of implementing element.Transformer because
// elements written to the stream are automatically closed and the stream
// header should not close until the stream is closed.
func (s StreamHeader) WriteBytes() []byte {
	var buf bytes.Buffer
	buf.WriteString("<stream:stream")
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("to='%s'", s.To))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("from='%s'", s.From))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("version='%s'", s.Version))
	buf.WriteByte(' ')
	if s.Lang != "" {
		buf.WriteString(fmt.Sprintf("xml:lang='%s'", s.Lang))
		buf.WriteByte(' ')
	}
	if s.Namespace != "" {
		buf.WriteString(fmt.Sprintf("xmlns='%s'", s.Namespace))
		buf.WriteByte(' ')
	}
	buf.WriteString(fmt.Sprintf("xmlns:stream='%s'", namespace.Stream))
	buf.WriteByte('>')
	return buf.Bytes()
}
