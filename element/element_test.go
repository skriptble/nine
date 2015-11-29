package element

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestElementWrite(t *testing.T) {
	t.Parallel()

	var want, got string
	var n int64
	var err error
	var el Element
	var buf bytes.Buffer
	var b []byte

	// Should be able to write element into an io.Writer.
	el = Element{
		Space: "namespace",
		Tag:   "foo",
		Attr: []Attr{
			{Space: "foo", Key: "bar", Value: "val"},
			{Key: "bar2", Value: "val2"},
		},
		Child: []Token{
			Element{
				Tag:   "foobar",
				Child: []Token{CharData{Data: "Random Data Whee"}},
			},
		},
	}
	want = `<namespace:foo foo:bar="val" bar2="val2">`
	want += `<foobar>Random Data Whee</foobar></namespace:foo>`
	n, err = el.WriteTo(&buf)
	if err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	if len(want) != int(n) {
		t.Error("Incorrect number of bytes written to io.Writer.")
		t.Errorf("\nWant:%d\nGot :%d", len(want), got)
	}
	got = buf.String()
	if got != want {
		t.Error("Should be able to write element into an io.Writer.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}

	// Should be able to write element into a slice of bytes.
	b = el.WriteBytes()
	got = string(b)
	if want != got {
		t.Error("Should be able to write element into a slice of bytes.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestElementWriterError(t *testing.T) {
	t.Parallel()

	var want, got error
	var writer io.Writer
	var el Element
	var n int64

	el = Element{Tag: "foo"}

	// Should return error from underlying io.Writer.
	want = errors.New("io.Writer error")
	writer = errWriter{err: want}
	n, got = el.WriteTo(writer)
	if n != 0 {
		t.Error("Incorrect number of bytes written to io.Writer.")
		t.Errorf("\nWant:%d\nGot :%d", 0, n)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Should return error from underlying io.Writer.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestElementText(t *testing.T) {
	t.Parallel()

	var want, got string
	var el Element

	// Should return empty text if there are no children.
	el = Element{Tag: "foo"}
	want = ""
	got = el.Text()
	if want != got {
		t.Error("Should return empty text if there are no children.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}

	// Should return empty text if no children are CharData.
	el = Element{Tag: "foo", Child: []Token{Element{Tag: "bar"}}}
	want = ""
	got = el.Text()
	if want != got {
		t.Error("Should return empty text if there are no children.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}

	// Should return text if first element is CharData.
	el = Element{Tag: "foo", Child: []Token{CharData{Data: "barbaz"}}}
	want = "barbaz"
	got = el.Text()
	if want != got {
		t.Error("Should return empty text if there are no children.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestElementSetText(t *testing.T) {
	t.Parallel()

	var el, want, got Element

	// Should be able to set text on an element with children, with first child CharData.
	want = Element{Tag: "foo", Child: []Token{CharData{Data: "foobarbaz"}, Element{Tag: "bar"}}}
	el = Element{Tag: "foo", Child: []Token{CharData{Data: "wrongdata"}, Element{Tag: "bar"}}}
	got = el.SetText("foobarbaz")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to set text on an element with children, with first child CharData.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
	// Should be able to set text on an element with children.
	want = Element{Tag: "foo", Child: []Token{CharData{Data: "foobarbaz"}, Element{Tag: "bar"}}}
	el = Element{Tag: "foo", Child: []Token{Element{Tag: "bar"}}}
	got = el.SetText("foobarbaz")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to set text on an element with children.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should be able to set text on an element with no children.
	want = Element{Tag: "foo", Child: []Token{CharData{Data: "foobarbaz"}}}
	el = Element{Tag: "foo"}
	got = el.SetText("foobarbaz")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to set text on an element with no children.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

}

func TestElementSelectAttr(t *testing.T) {
	t.Parallel()

	var want, got Attr
	var el Element

	// Should be able to get Attr which exists on element.
	want = Attr{Key: "baz", Value: "quux"}
	el = Element{Tag: "foo", Attr: []Attr{want}}
	got = el.SelectAttr("baz")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to get Attr which exists on element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should get NoAttrExists when the attribute key does not exist on element.
	got = el.SelectAttr("doesn't exist")
	if got != NoAttrExists {
		t.Error("Should get NoAttrexists when the attribute key does not exists on element.")
		t.Errorf("\nWant:%+v\nGot :%+v", NoAttrExists, got)
	}
}

func TestElementSelectAttrValue(t *testing.T) {
	t.Parallel()

	var want, got string
	var el Element

	// Should be able to get Attr value for Attr which exists on element.
	want = "quux"
	el = Element{Tag: "foo", Attr: []Attr{{Key: "baz", Value: "quux"}}}
	got = el.SelectAttrValue("baz", "wrong")
	if want != got {
		t.Error("Should be able to get Attr value for Attr which exists on element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should get default value for Attr which doesn't exist on element.
	want = "default value wheee"
	got = el.SelectAttrValue("doesn't exist", want)
	if want != got {
		t.Error("Should get default value for Attr which doesn't exist on element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestElementChildElements(t *testing.T) {
	t.Parallel()

	var want, got []Element
	var el Element

	// Should return elements if the element has child elements.
	el = Element{Tag: "foo",
		Child: []Token{
			CharData{Data: "Random Data"},
			Element{Tag: "bar"},
			Element{Space: "namespace", Tag: "baz"},
		},
	}
	want = []Element{{Tag: "bar"}, {Space: "namespace", Tag: "baz"}}
	got = el.ChildElements()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should return elements if the element has child elements.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should return no elements if the element has no child elements.
	el = Element{Tag: "foo", Child: []Token{CharData{Data: "Random Data"}}}
	want = []Element{}
	got = el.ChildElements()
	if len(got) != len(want) {
		t.Error("Should return no elements if the element hasno child elements.")
		t.Errorf("\nWant:%d\nGot :%d", len(want), len(got))
	}
}

func TestSelectElement(t *testing.T) {
	t.Parallel()

	var el, want, got Element

	// Should return child element if the child element exists.
	el = Element{Tag: "foo", Child: []Token{
		Element{Tag: "bar"},
		Element{Space: "namespace", Tag: "bar"},
	}}
	want = Element{Tag: "bar"}
	got = el.SelectElement("bar")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should return child element if the child element exists.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should properly decompose tag string to retrieve child element.
	want = Element{Space: "namespace", Tag: "bar"}
	got = el.SelectElement("namespace:bar")
	if !reflect.DeepEqual(want, got) {
		t.Error("Should propery decompose tag string to retrieve child element.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Should return NoElementExists if the child element doesn't exist.
	got = el.SelectElement("doesn't exist")
	if !reflect.DeepEqual(NoElementExists, got) {
		t.Error("Should return NoElementExists if the child element doesn't exist.")
		t.Errorf("\nWant:%+v\nGot :%+v", NoElementExists, got)
	}
}

func TestDecompose(t *testing.T) {
	t.Parallel()

	var space, key string

	// Should decompose non-namespaced tag into empty space with string as key.
	space, key = decompose("nonnamedspacedtagfoo")
	if space != "" {
		t.Error("Should decompose non-namspaced tag into empty space with string as key")
		t.Errorf("\nWant:%s\nGot :%s", "", space)
	}
	if key != "nonnamedspacedtagfoo" {
		t.Error("Should decompose non-namspaced tag into empty space with string as key")
		t.Errorf("\nWant:%s\nGot :%s", "nonnamespacedtagfoo", key)
	}

	// Should decompose namespaced tag into tag and key.
	space, key = decompose("namespaced:tagfoo")
	if space != "namespaced" {
		t.Error("Should decompose namespaced tag into tag and key")
		t.Errorf("\nWant:%s\nGot :%s", "namespaced", space)
	}
	if key != "tagfoo" {
		t.Error("Should decompose namespaced tag into tag and key")
		t.Errorf("\nWant:%s\nGot :%s", "tagfoo", key)
	}
}

type errWriter struct{ err error }

func (ew errWriter) Write(_ []byte) (int, error) { return 0, ew.err }
