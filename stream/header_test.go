package stream

import (
	"errors"
	"reflect"
	"testing"

	"github.com/skriptble/nine/element"
)

func TestNewHeader(t *testing.T) {
	t.Parallel()

	var want, got Header
	var el element.Element
	var err error

	// Should be able to create StreamHeader from Element.
	el = element.Element{Space: "stream", Tag: "stream",
		Attr: []element.Attr{
			{Key: "to", Value: "foo"},
			{Key: "from", Value: "bar"},
			{Key: "id", Value: "randomid"},
			{Space: "xml", Key: "lang", Value: "en"},
			{Key: "version", Value: "1.0"},
			{Key: "xmlns", Value: "jabber:client"},
		},
	}
	want = Header{
		To: "foo", From: "bar",
		ID:   "randomid",
		Lang: "en", Version: "1.0",
		Namespace: "jabber:client",
	}
	got, err = NewHeader(el)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to create StreamHeader from Element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestNewHeaderError(t *testing.T) {
	t.Parallel()

	var want, got error
	var el element.Element

	// Should return error if the element is not a stream header.
	el = element.Element{Space: "not", Tag: "astream"}
	want = errors.New("Element is not <stream:stream> it is a <not:astream>")
	_, got = NewHeader(el)
	if want.Error() != got.Error() {
		t.Error("Should return error if the element is not a stream header.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestHeaderWriteBytes(t *testing.T) {
	t.Parallel()

	var want, got []byte
	var strm Header

	// Should be able to write StreamHeader to bytes.
	strm = Header{
		To: "foo", From: "bar",
		ID:   "randomid",
		Lang: "en", Version: "1.0",
		Namespace: "jabber:client",
	}
	want = []byte("<stream:stream to='foo' from='bar' id='randomid' version='1.0' xml:lang='en' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>")
	got = strm.WriteBytes()

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to write StreamHeader to bytes.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}
