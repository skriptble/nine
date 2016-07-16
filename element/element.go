package element

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// NoAttrExists is returned when SelectAttr is called with a key that no
// attributes of the element contain.
var NoAttrExists = Attr{}

// NoElementExists is returned when SelectElement is called with a tag that no
// child elements of the element contain.
var NoElementExists = Element{}

// Token is an interface implemented by things that can be a child of an
// element.
type Token interface {
	write(w *bufio.Writer)
}

// Element represents an XML element.
type Element struct {
	Space, Tag string
	Namespaces map[string]string
	Attr       []Attr
	Child      []Token
}

// New creates a new element using the tag. The tag is decomposed into its
// space and tag if it contains a colon.
func New(tag string) Element {
	space, tag := decompose(tag)
	return Element{Space: space, Tag: tag, Namespaces: make(map[string]string)}
}

// AddAttr creates an Attr and appends it to the element. The key is decomposed
// into a space and key if it contains a colon:
func (e Element) AddAttr(key, value string) Element {
	space, key := decompose(key)
	attr := Attr{Key: key, Space: space, Value: value}
	e.Attr = append(e.Attr, attr)
	return e
}

// Transformer is an interface implemented by types that can transform
// themselves into an Element.
type Transformer interface {
	TransformElement() Element
}

// Attr represents an attribute of an XML element.
type Attr struct {
	Space, Key string
	Value      string
}

// CharData represents the character data of an XML element.
type CharData struct {
	Data       string
	whitespace bool
}

// WriteTo implements io.WriterTo
func (e Element) WriteTo(w io.Writer) (n int64, err error) {
	cw := newCountWriter(w)
	b := bufio.NewWriter(cw)
	e.write(b)
	err = b.Flush()
	return cw.bytes, err
}

// WriteBytes serializes the Element into a slice of bytes.
func (e Element) WriteBytes() []byte {
	var buf bytes.Buffer
	e.WriteTo(&buf)
	return buf.Bytes()
}

// String implements the fmt.Stringer interface.
func (e Element) String() string {
	return string(e.WriteBytes())
}

// Text returns the text data of the element.
func (e Element) Text() string {
	if len(e.Child) == 0 {
		return ""
	}
	if cd, ok := e.Child[0].(CharData); ok {
		return cd.Data
	}

	return ""
}

// SetText sets the text data of the element.
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

// SelectAttr returns the attribute with the given key. The key can include a
// namespace. If no attribute exists NoAttrExists is returned.
func (e Element) SelectAttr(key string) Attr {
	space, skey := decompose(key)
	for _, a := range e.Attr {
		if space == a.Space && skey == a.Key {
			return a
		}
	}
	return NoAttrExists
}

// SelectAttrValue returns the attribute with the given key. The key can
// include a namespace. If no attribute exists, the default value is returned.
func (e Element) SelectAttrValue(key, dflt string) string {
	space, key := decompose(key)
	for _, a := range e.Attr {
		if space == a.Space && key == a.Key {
			return a.Value
		}
	}

	return dflt
}

// ChildElements returns all children of the element who are elements
// themselves.
func (e Element) ChildElements() (elements []Element) {
	for _, t := range e.Child {
		if c, ok := t.(Element); ok {
			elements = append(elements, c)
		}
	}
	return
}

// AddChild adds the element to the parent's children.
func (e Element) AddChild(el Element) Element {
	e.Child = append(e.Child, el)
	return e
}

// SelectElement returns the element with the given tag. The tag can include
// a namespace. If there is no child element with the tag, NoElementExists is
// returned.
func (e Element) SelectElement(tag string) Element {
	space, tag := decompose(tag)
	for _, t := range e.Child {
		if c, ok := t.(Element); ok && equalSpace(space, c.Space) && tag == c.Tag {
			return c
		}
	}
	return NoElementExists
}

// MatchNamespace returns true if the namespace for this element matches the
// namespace provided.
func (e Element) MatchNamespace(ns string) bool {
	elNS, ok := e.Namespaces[e.Space]
	if !ok {
		return false
	}
	return elNS == ns
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
	w.Write([]byte{'=', '\''})
	w.WriteString(escape(a.Value))
	w.WriteByte('\'')
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

func equalSpace(a, b string) bool {
	if a == "" {
		return true
	}

	return a == b
}

// countWriter is used to count the number of bytes written to an underlying
// io.Writer.
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
