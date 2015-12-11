package sasl

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

type Handler struct {
	mechs map[string]Mechanism
}

type Mechanism interface {
	Authenticate(data string, props stream.Properties) ([]element.Element, stream.Properties)
}

func (h Handler) HandleFeature(props stream.Properties) stream.Properties {
	if props.Status&stream.Auth != 0 {
		return props
	}
	mechs := element.SASLMechanisms
	for name := range h.mechs {
		mechs = mechs.AddChild(element.New("mechanism").SetText(name))
	}
	props.Features = append(props.Features, mechs)
	return props
}

func (h Handler) HandleElement(el element.Element, props stream.Properties) (
	[]element.Element, stream.ElementFSM, stream.Properties) {
	var elems []element.Element
	switch el.Tag {
	case "auth":
		mechName := el.SelectAttrValue("mechanism", "")
		mech, ok := h.mechs[mechName]
		if !ok {
			// TODO(skriptble): Return a stream error
			break
		}
		data := el.Text()
		log.Println("Authenticating")
		elems, props = mech.Authenticate(data, props)
	case "response":
	}
	return elems, h, props
}

func NewHandler(mechs map[string]Mechanism) Handler {
	return Handler{mechs: mechs}
}

type PlainMech struct {
}

func (pm PlainMech) Authenticate(data string, props stream.Properties) ([]element.Element, stream.Properties) {
	// TODO(skriptble): Implement for real to spec. Just trying to get things
	// working.
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return []element.Element{element.MalformedRequest}, props
	}

	res := strings.Split(string(decoded), "\000")
	if len(res) != 3 {
		return []element.Element{element.MalformedRequest}, props
	}
	props.Header.To = res[1] + "@localhost"
	props.Status = props.Status | stream.Restart | stream.Auth
	return []element.Element{element.SASLSuccess}, props
}
