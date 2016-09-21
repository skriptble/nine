package stream

import (
	"errors"
	"reflect"
	"testing"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/jid"
)

func TestIQMuxHandle(t *testing.T) {
	t.Parallel()

	var im = NewIQMux()
	var ies []iqEntry
	var err, wantErr error

	// Handle should return if err is set
	wantErr = errors.New("Already Set Error")
	im.err = wantErr
	im = im.Handle("", "", "", nil)
	if im.err != wantErr {
		t.Error("Handle should return if err is set")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, im.err)
	}

	// Handle should return ErrSpaceTagTypeEmpty if the space is empty
	im = NewIQMux()
	im = im.Handle("foo", "", "", nil)
	err = im.Err()
	wantErr = ErrSpaceTagTypeEmpty
	if err != wantErr {
		t.Error("Handle should return ErrSpaceTagTypeEmpty if the space is empty.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return ErrSpaceTagTypeEmpty if the tag is empty
	im = NewIQMux()
	im = im.Handle("", "foo", "", nil)
	err = im.Err()
	wantErr = ErrSpaceTagTypeEmpty
	if err != wantErr {
		t.Error("Handle should return ErrSpaceTagTypeEmpty if the tag is empty.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return ErrSpaceTagTypeEmpty if the type is empty
	im = NewIQMux()
	im = im.Handle("", "", "foo", nil)
	err = im.Err()
	wantErr = ErrSpaceTagTypeEmpty
	if err != wantErr {
		t.Error("Handle should return ErrSpaceTagTypeEmpty if the type is empty.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return ErrNilIQHandler if the IQHandler is nil
	im = NewIQMux()
	im = im.Handle("foo", "bar", "set", nil)
	err = im.Err()
	wantErr = ErrNilIQHandler
	if err != wantErr {
		t.Error("Handle should return ErrNilIQHandler if the IQHandler is nil.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should return an error if a space, tag, and type combination is
	// registered more than once
	im = NewIQMux()
	im = im.Handle("foo", "bar", "set", stubIQHandler{}).
		Handle("foo", "bar", "set", stubIQHandler{})
	err = im.Err()
	wantErr = errors.New("Multiple registrations for type set and tag <foo:bar>")
	if err.Error() != wantErr.Error() {
		t.Error("Handle should return an error if a space, tag, and type combination is registered more than once.")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Handle should add an ElementHandler for a given space and tag to its set
	// of ElementHandler entries.
	im = NewIQMux().Handle("foo", "bar", "set", stubIQHandler{})
	ies = []iqEntry{{space: "foo", tag: "bar", stanzaType: "set", h: stubIQHandler{}}}
	if !reflect.DeepEqual(ies, im.handlers) {
		t.Error("Handle should add an IQHandler for a given space, tag, and type to its set of IQHandler entries.")
		t.Errorf("\nWant:%+v\nGot :%+v", ies, im.handlers)
	}
}

func TestIQMuxErr(t *testing.T) {
	t.Parallel()

	// Err should return error on ElementMux
	want := errors.New("IQMux Error")
	imux := IQMux{err: want}
	got := imux.Err()
	if want != got {
		t.Error("Err should return error on IQMux.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestIQMuxHandler(t *testing.T) {
	t.Parallel()

	var want, got IQHandler
	var entry iqEntry
	var im IQMux
	var el element.Element

	want = stubIQHandler{}
	el = element.New("handler").AddNamespace("", "stub")
	entry = iqEntry{space: "stub", tag: "handler", stanzaType: "get", h: want}
	im.handlers = append(im.handlers, entry)
	// Handler should return a matching IQHandler
	got = im.Handler(el, "get")
	if !reflect.DeepEqual(want, got) {
		t.Error("Handler should return a matching IQHandler.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Handler should return ServiceUnavailable if no handlers match
	want = ServiceUnavailable{}
	el = element.New("bar").AddAttr("xmlns", "foo")
	got = im.Handler(el, "set")
	if got != want {
		t.Error("Handler should return a ServiceUnavailable if no handlers match.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestIQMuxHandleElement(t *testing.T) {
	t.Parallel()

	var im IQMux
	var props Properties
	var want, got []element.Element
	var hdlr IQHandler

	props.To = "foo@bar.baz"
	to := jid.New("foo@bar.baz")
	from := jid.New("quux@foo.bar")
	iq := stanza.NewIQResult(to, from, "random-id", stanza.IQSet)

	// HandleElement should call the given handler and return the stanzas as
	// elements
	props.Status = props.Status | Bind
	iq.Children = []element.Element{element.New("bar").AddAttr("xmlns", "foo")}
	hdlr = stubIQHandler{iq: iq}
	im = NewIQMux().Handle("foo", "bar", string(stanza.IQSet), hdlr)
	if im.Err() != nil {
		t.Errorf("An unexpected error occured: %s", im.Err())
	}
	want = []element.Element{iq.TransformElement()}
	got, _, _, _ = im.HandleElement(iq.TransformElement())
	if !reflect.DeepEqual(want, got) {
		t.Error("HandleElement should call the correct handler and return the stanzas as elements.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestServiceUnavailable(t *testing.T) {
	t.Parallel()

	// ServiceUnavailable should return properties unaltered and a single
	// element of unsupported-stanza-type
	var want, got []stanza.Stanza
	var iq stanza.IQ

	st := stanza.NewIQError(iq, element.Stanza.ServiceUnavailable).TransformStanza()
	want = []stanza.Stanza{st}
	su := ServiceUnavailable{}
	got, _, _, _ = su.HandleIQ(iq)

	if !reflect.DeepEqual(want, got) {
		t.Error("ServiceUnavailable should return a single stanza of service-unavailable.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

type stubIQHandler struct{ iq stanza.IQ }

func (sih stubIQHandler) HandleIQ(_ stanza.IQ) ([]stanza.Stanza, StateChange, bool, bool) {
	return []stanza.Stanza{sih.iq.TransformStanza()}, nil, false, false
}

func (sih stubIQHandler) Update(_, _ string) {}
