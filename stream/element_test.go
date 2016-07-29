package stream

import (
	"errors"
	"reflect"
	"testing"

	"github.com/skriptble/nine/element"
)

func TestElementMuxHandle(t *testing.T) {
	t.Parallel()

	var em = NewElementMux()
	var ees []elementEntry
	var err, wantErr error

	// Handle should return if err is set
	wantErr = errors.New("Already Set Error")
	em.err = wantErr
	em = em.Handle("", "", nil)
	if em.err != wantErr {
		t.Error("Handle should return if err is set")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, em.err)
	}

	// Handle should return ErrEmptySpaceTag if the space is empty
	em = NewElementMux()
	em = em.Handle("", "foo", nil)
	err = em.Err()
	wantErr = ErrEmptySpaceTag
	if err != wantErr {
		t.Error("Handle should return ErrEmptySpaceTag if the space is empty.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return ErrEmptySpaceTag if the tag is empty
	em = NewElementMux()
	em = em.Handle("foo", "", nil)
	err = em.Err()
	wantErr = ErrEmptySpaceTag
	if err != wantErr {
		t.Error("Handle should return ErrEmptySpaceTag if the tag is empty.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return ErrNilElementHandler if the ElementHandler is nil
	em = NewElementMux()
	em = em.Handle("foo", "bar", nil)
	err = em.Err()
	wantErr = ErrNilElementHandler
	if err != wantErr {
		t.Error("Handle should return ErrNilElementHandler if the ElementHandler is nil.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return an error if a space and tag combination is
	// registered more than once
	em = NewElementMux()
	em = em.Handle("foo", "bar", Blackhole{}).Handle("foo", "bar", Blackhole{})
	err = em.Err()
	wantErr = errors.New("stream: multiple registrations for <foo:bar>")
	if err.Error() != wantErr.Error() {
		t.Error("Handle should return an error if the space and tag combination is registered more than once.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should add an ElementHandler for a given space and tag to its set
	// of ElementHandler entries.
	em = NewElementMux().Handle("foo", "bar", Blackhole{})
	ees = []elementEntry{{space: "foo", tag: "bar", h: Blackhole{}}}
	if !reflect.DeepEqual(ees, em.m) {
		t.Error("Handle should add an ElementHandler for a given space and tag to its set of ElementHandler entries.")
		t.Errorf("\nWant:%+v\nGot :%+v", ees, em.m)
	}
}

func TestElementMuxErr(t *testing.T) {
	t.Parallel()

	// Err should return error on ElementMux
	want := errors.New("ElementMux Error")
	emux := ElementMux{err: want}
	got := emux.Err()
	if want != got {
		t.Error("Err should return error on ElementMux.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestElementMuxHandleElement(t *testing.T) {
	t.Parallel()

	var sh = stubHandler{}
	var el = element.New("bar").AddAttr("xmlns", "foo")
	var entry = elementEntry{space: "foo", tag: "bar", h: &sh}
	var em = NewElementMux()
	em.m = append(em.m, entry)

	// HandleElement should call the handler for a matching element.
	em.HandleElement(el, Properties{})
	if !sh.called {
		t.Error("HandleElement should call the handler for a matching element.")
	}
}

func TestElementMuxHandler(t *testing.T) {
	t.Parallel()

	var want, got ElementHandler
	var entry elementEntry
	var em ElementMux
	var el element.Element

	want = Blackhole{}
	entry = elementEntry{space: "black", tag: "hole", h: want}
	el = element.New("hole").AddAttr("xmlns", "black")
	em.m = append(em.m, entry)
	// Handler should return a matching ElementHandler
	got = em.Handler(el)
	if got != want {
		t.Error("Handler should return a matching ElementHandler.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Handler should return UnsupportedStanza if no handlers match
	want = UnsupportedStanza{}
	el = element.New("bar").AddAttr("xmlns", "foo")
	got = em.Handler(el)
	if got != want {
		t.Error("Handler should return a UnsupportedStanza if no handlers match.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestUnsupportedStanza(t *testing.T) {
	t.Parallel()

	// UnsupportedStanza should return properties with the status closed
	// and a single element of unsupported-stanza-type
	var want, got []element.Element
	var el element.Element
	var props Properties

	want = []element.Element{element.StreamError.UnsupportedStanzaType}
	el = element.New("foo")
	us := UnsupportedStanza{}
	got, props = us.HandleElement(el, Properties{})
	if props.Status != Closed {
		t.Error("UnsupportedStanza should return properties witht he status closed.")
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("UnsupportedStanza should return a single element of unsupported-stanza-type.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestBlackhole(t *testing.T) {
	t.Parallel()

	var want, got Properties
	var el element.Element
	var els []element.Element
	// Blackhole should return properties unaltered and return no elements
	want.Status = (Auth | Bind)
	want.To = "foo"
	want.From = "bar"
	bh := Blackhole{}
	el = element.New("foo")
	els, got = bh.HandleElement(el, want)
	if !reflect.DeepEqual(els, []element.Element{}) {
		t.Error("Blackhole should return no elements.")
		t.Errorf("Wanted no elements, got %+v", els)
	}
	if !reflect.DeepEqual(want, got) {
		t.Error("Blackhole should return properties unaltered.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

type stubHandler struct{ called bool }

func (sh *stubHandler) HandleElement(_ element.Element, _ Properties) ([]element.Element, Properties) {
	sh.called = true
	return []element.Element{}, Properties{}
}
