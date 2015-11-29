package element

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewStreamHeader(t *testing.T) {
	t.Parallel()

	var want, got StreamHeader
	var el Element
	var err error

	// Should be able to create StreamHeader from Element.
	el = Element{Space: "stream", Tag: "stream",
		Attr: []Attr{
			{Key: "to", Value: "foo"},
			{Key: "from", Value: "bar"},
			{Space: "xml", Key: "lang", Value: "en"},
			{Key: "version", Value: "1.0"},
			{Key: "xmlns", Value: "jabber:client"},
		},
	}
	want = StreamHeader{
		To: "foo", From: "bar",
		Lang: "en", Version: "1.0",
		Namespace: "jabber:client",
	}
	got, err = NewStreamHeader(el)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to create StreamHeader from Element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestNewStreamHeaderError(t *testing.T) {
	t.Parallel()

	var want, got error
	var el Element

	// Should return error if the element is not a stream header.
	el = Element{Space: "not", Tag: "astream"}
	want = errors.New("Element is not <stream:stream> it is a <not:astream>")
	_, got = NewStreamHeader(el)
	if want.Error() != got.Error() {
		t.Error("Should return error if the element is not a stream header.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestStreamHeaderWriteBytes(t *testing.T) {
	t.Parallel()

	var want, got []byte
	var strm StreamHeader

	// Should be able to write StreamHeader to bytes.
	strm = StreamHeader{
		To: "foo", From: "bar",
		Lang: "en", Version: "1.0",
		Namespace: "jabber:client",
	}
	want = []byte("<stream:stream to='foo' from='bar' version='1.0' xml:lang='en' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>")
	got = strm.WriteBytes()

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to write StreamHeader to bytes.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}
