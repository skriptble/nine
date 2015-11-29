package stream

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/skriptble/nine/element"
)

func TestStreamNext(t *testing.T) {
	t.Parallel()

	var want, got element.Element
	var err error

	// Should be able to retrieve elements from stream.
	xmlstr := "<?xml version='1.0'?>"
	xmlstr += "<stream:stream>"
	xmlstr += `<foo baz="quux" namespace:baz="quux"><bar>Hello World</bar>`
	xmlstr += `<bar>Goodbye World</bar>`
	xmlstr += `<!-- whee commenting -->`
	xmlstr += `<namespaced:bar>Namespace World</namespaced:bar>`
	xmlstr += `</foo><baz:quux></baz:quux>`
	xmlstr += "</stream:stream>"
	buf := bytes.NewBufferString(xmlstr)
	strm := New(buf)
	got, err = strm.Next()
	if err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	want = element.Element{Space: "stream", Tag: "stream"}
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to retrieve elements from stream")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	got, err = strm.Next()
	if err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	want = element.Element{
		Tag: "foo",
		Attr: []element.Attr{
			{Key: "baz", Value: "quux"},
			{Space: "namespace", Key: "baz", Value: "quux"},
		},
		Child: []element.Token{
			element.Element{
				Tag:   "bar",
				Child: []element.Token{element.CharData{Data: "Hello World"}},
			},
			element.Element{
				Tag:   "bar",
				Child: []element.Token{element.CharData{Data: "Goodbye World"}},
			},
			element.Element{
				Space: "namespaced",
				Tag:   "bar",
				Child: []element.Token{element.CharData{Data: "Namespace World"}},
			},
		},
	}
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to retrieve elements from stream")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	got, err = strm.Next()
	if err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	want = element.Element{Space: "baz", Tag: "quux"}
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to retrieve elements from stream")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	_, err = strm.Next()
	if err != StreamClosed {
		t.Error("Expected stream to be closed after reading </stream:stream> tag.")
	}
}

func TestStreamNextError(t *testing.T) {
	t.Parallel()

	var got error
	var buf *bytes.Buffer
	var strm Stream

	brokenxml1 := "</foo>"
	brokenxml2 := "<foo><bar><baz></bar></foo>"

	// Should return error from top level element broken xml
	buf = bytes.NewBufferString(brokenxml1)
	strm = New(buf)
	_, got = strm.Next()
	if got == nil {
		t.Error("Should return error for broken xml (top level element)")
	}

	buf = bytes.NewBufferString(brokenxml2)
	strm = New(buf)
	_, got = strm.Next()
	if got == nil {
		t.Error("Should return error for broken xml (child level element)")
	}
}

func TestStreamWrite(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var want, got []byte
	var strm Stream

	strm = New(&buf)

	// Should be able to write bytes to stream.
	want = []byte("Testing Writing of arbitrary bytes")
	n, err := strm.Write(want)
	if err != nil {
		t.Errorf("Unexecpted error: %s", err)
	}
	if len(want) != n {
		t.Error("Incorrect number of bytes written to stream.")
		t.Errorf("\nWant:%d\nGot :%d", len(want), n)
	}
	got = buf.Bytes()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should bea ble to write arbitrary bytes to stream.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestStreamWriteError(t *testing.T) {
	t.Parallel()

	var want, got error
	var strm Stream

	// Errors from writing bytes to stream should be returned.
	want = errors.New("Error from underlying stream")
	strm = New(errReadWriter{want})
	_, got = strm.Write([]byte("foo bar"))
	if want != got {
		t.Error("Errors from underlying io.ReadWriter should be returned from stream.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestStreamWriteElement(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var want, got []byte
	var strm Stream
	var el = element.Element{Tag: "foo"}

	// Should be able to write element to stream
	want = []byte("<foo/>")
	strm = New(&buf)

	n, err := strm.WriteElement(el)
	if err != nil {
		t.Errorf("Unexecpted error: %s", err)
	}
	if len(want) != n {
		t.Error("Incorrect number of bytes written to stream.")
		t.Errorf("\nWant:%d\nGot :%d", len(want), n)
	}
	got = buf.Bytes()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should bea ble to write arbitrary bytes to stream.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestStreamWriteElementError(t *testing.T) {
	t.Parallel()

	var want, got error
	var strm Stream
	var el = element.Element{Tag: "foo"}

	// Errors from writing bytes to stream should be returned.
	want = errors.New("Error from underlying stream")
	strm = New(errReadWriter{want})
	_, got = strm.WriteElement(el)
	if want != got {
		t.Error("Errors from underlying io.ReadWriter should be returned from stream.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

type errReadWriter struct {
	err error
}

func (erw errReadWriter) Write(p []byte) (n int, err error) {
	return 0, erw.err
}

func (erw errReadWriter) Read(p []byte) (n int, err error) {
	return 0, erw.err
}
