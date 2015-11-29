package element

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

var NoAttrExists = Attr{}
var NoElementExists = Element{}

type Token interface {
	write(w *bufio.Writer)
}

type Element struct {
	Space, Tag string
	Attr       []Attr
	Child      []Token
}

type Attr struct {
	Space, Key string
	Value      string
}

type CharData struct {
	Data       string
	whitespace bool
}

func (e Element) WriteTo(w io.Writer) (n int64, err error) {
	cw := newCountWriter(w)
	b := bufio.NewWriter(cw)
	e.write(b)
	err = b.Flush()
	return cw.bytes, err
}

func (e Element) WriteBytes() []byte {
	var buf bytes.Buffer
	e.WriteTo(&buf)
	return buf.Bytes()
}

func (e Element) Text() string {
	if len(e.Child) == 0 {
		return ""
	}
	if cd, ok := e.Child[0].(CharData); ok {
		return cd.Data
	}

	return ""
}

func (e Element) SetText(text string) Element {
	if len(e.Child) > 0 {
		if cd, ok := e.Child[0].(CharData); ok {
			cd.Data = text
			e.Child[0] = cd
			return e
		}
	}
	e.Child = append(e.Child, nil)
	copy(e.Child[1:], e.Child[0:])
	e.Child[0] = CharData{Data: text}
	return e
}

func (e Element) SelectAttr(key string) Attr {
	space, skey := decompose(key)
	for _, a := range e.Attr {
		if space == a.Space && skey == a.Key {
			return a
		}
	}
	return NoAttrExists
}

func (e Element) SelectAttrValue(key, dflt string) string {
	space, key := decompose(key)
	for _, a := range e.Attr {
		if space == a.Space && key == a.Key {
			return a.Value
		}
	}

	return dflt
}

func (e Element) ChildElements() (elements []Element) {
	for _, t := range e.Child {
		if c, ok := t.(Element); ok {
			elements = append(elements, c)
		}
	}
	return
}

func (e Element) SelectElement(tag string) Element {
	space, tag := decompose(tag)
	for _, t := range e.Child {
		if c, ok := t.(Element); ok && space == c.Space && tag == c.Tag {
			return c
		}
	}
	return NoElementExists
}

func (e Element) write(w *bufio.Writer) {
	w.WriteByte('<')
	if e.Space != "" {
		w.WriteString(e.Space)
		w.WriteByte(':')
	}
	w.WriteString(e.Tag)
	for _, a := range e.Attr {
		w.WriteByte(' ')
		a.write(w)
	}
	if len(e.Child) > 0 {
		w.WriteByte('>')
		for _, c := range e.Child {
			c.write(w)
		}
		w.Write([]byte{'<', '/'})
		if e.Space != "" {
			w.WriteString(e.Space)
			w.WriteByte(':')
		}
		w.WriteString(e.Tag)
		w.WriteByte('>')
	} else {
		w.Write([]byte{'/', '>'})
	}
}

func (a Attr) write(w *bufio.Writer) {
	if a.Space != "" {
		w.WriteString(a.Space)
		w.WriteByte(':')
	}
	w.WriteString(a.Key)
	w.Write([]byte{'=', '"'})
	w.WriteString(escape(a.Value))
	w.WriteByte('"')
}

func (c CharData) write(w *bufio.Writer) {
	w.WriteString(escape(c.Data))
}

func decompose(str string) (space, key string) {
	strs := strings.SplitN(str, ":", 2)
	if (len(strs)) < 2 {
		return "", str
	}

	return strs[0], strs[1]
}

var xmlReplacer = strings.NewReplacer(
	"<", "&lt;",
	">", "&gt;",
	"&", "&amp;",
	"'", "&apos;",
	`"`, "&quot;",
)

func escape(s string) string {
	return xmlReplacer.Replace(s)
}

type countWriter struct {
	w     io.Writer
	bytes int64
}

func newCountWriter(w io.Writer) *countWriter {
	return &countWriter{w: w}
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	b, err := cw.w.Write(p)
	cw.bytes += int64(b)
	return b, err
}
