package stream

import (
	"bytes"
	"fmt"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/namespace"
)

// Header represents an XML stream element.
type Header struct {
	To, From  string
	ID        string
	Lang      string
	Version   string
	Namespace string
}

// NewHeader attempts to transform the Element into a Stream. Returns an error
// if the element is not a stream element.
func NewHeader(el element.Element) (strm Header, err error) {
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
		case "id":
			strm.ID = attr.Value
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
func (h Header) WriteBytes() []byte {
	var buf bytes.Buffer
	buf.WriteString("<stream:stream")
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("to='%s'", h.To))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("from='%s'", h.From))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("id='%s'", h.ID))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprintf("version='%s'", h.Version))
	buf.WriteByte(' ')
	if h.Lang != "" {
		buf.WriteString(fmt.Sprintf("xml:lang='%s'", h.Lang))
		buf.WriteByte(' ')
	}
	if h.Namespace != "" {
		buf.WriteString(fmt.Sprintf("xmlns='%s'", h.Namespace))
		buf.WriteByte(' ')
	}
	buf.WriteString(fmt.Sprintf("xmlns:stream='%s'", namespace.Stream))
	buf.WriteByte('>')
	return buf.Bytes()
}
