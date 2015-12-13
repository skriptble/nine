package features

import (
	"log"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/stream"
)

type Bind struct {
}

func NewBind() Feature {
	return Bind{}
}

// Negotiate implements the feature.Feature interface.
func (b Bind) Negotiate(el element.Element, s stream.Streamv0) (
	ns stream.Streamv0, restart, closeStream, retry bool) {
	// Ensure the element is an iq
	iq := stanza.TransformIQ(el)
	for _, child := range iq.Children {
		if child.Tag == "bind" && child.SelectAttrValue("xmlns", "") == namespace.Bind {
			resource := child.SelectElement("resource")
			log.Printf("%+v", resource)
			str := resource.Text()
			s.Header.To += "/" + str
		}
	}
	res := stanza.IQ{stanza.Stanza{ID: iq.ID, Type: "result"}}
	bindEl := element.Bind
	bindEl.Child = append(bindEl.Child, element.Element{
		Tag: "jid", Child: []element.Token{element.CharData{Data: s.Header.To}},
	})
	res.Children = append(res.Children, bindEl)
	s.WriteElement(res.TransformElement())
	return s, false, false, false
}

// TransformElement implements the feature.Feature interface.
func (b Bind) TransformElement() element.Element {
	return element.Bind
}
