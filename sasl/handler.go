package sasl

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/stream"
)

type Handler struct {
	mechs   map[string]Mechanism
	current Mechanism
}

// Mechanism is the interface implemented by SASL Mechanisms.
type Mechanism interface {
	// Authenticate is the method used when data is recieved. Elements returned
	// are written directly to the stream. The modified stream properties will
	// be assigned to the stream. If challenge is true, this mechanism will be
	// used for any subsequent response elements.
	Authenticate(data string, props stream.Properties) (elems []element.Element, p stream.Properties, challenge bool)
}

func (h *Handler) HandleFeature(props stream.Properties) stream.Properties {
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

func (h *Handler) HandleElement(el element.Element, props stream.Properties) (
	[]element.Element, stream.Properties) {
	var elems []element.Element
	var challenge bool
	switch el.Tag {
	case "auth":
		mechName := el.SelectAttrValue("mechanism", "")
		mech, ok := h.mechs[mechName]
		if !ok {
			elems = append(elems, element.SASLFailure.InvalidMechanism)
			break
		}
		data := el.Text()
		log.Println("Authenticating")
		elems, props, challenge = mech.Authenticate(data, props)
		if challenge {
			h.current = mech
		}
	case "response":
		if h.current == nil {
			el := element.SASLFailure.NotAuthorized.
				AddChild(element.New("text").SetText("Out of order SASL element"))
			elems = append(elems, el)
		}
		data := el.Text()
		elems, props, challenge = h.current.Authenticate(data, props)
		if !challenge {
			h.current = nil
		}
	}
	return elems, props
}

func NewHandler(mechs map[string]Mechanism) *Handler {
	return &Handler{mechs: mechs}
}

// PlainMech implements the plain SASL mechanism from RFC4616
type plainMech struct {
	auth PlainAuthenticator
}

// NewPlainMechanism creates a new SASL plain mechanism
func NewPlainMechanism(auth PlainAuthenticator) Mechanism {
	return plainMech{auth: auth}
}

// Authenticate implements the Mechanism interface for PlainMech
func (pm plainMech) Authenticate(data string, props stream.Properties) ([]element.Element, stream.Properties, bool) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return []element.Element{element.SASLFailure.MalformedRequest}, props, false
	}

	res := strings.Split(string(decoded), "\000")
	if len(res) != 3 {
		return []element.Element{element.SASLFailure.MalformedRequest}, props, false
	}
	identity, user, password := res[0], res[1], res[2]
	err = pm.auth.Authenticate(identity, user, password)
	// TODO: Handle different types of errors
	if err != nil {
		return []element.Element{element.SASLFailure.NotAuthorized}, props, false
	}
	if identity != "" {
		user = identity
	}

	// TODO: Add a way to determine the address of the server for the domain
	// part of the jid (do it better than this.)
	user += "@" + props.Domain
	j := jid.New(user)
	props.Header.To = j.String()
	props.Status = props.Status | stream.Restart | stream.Auth
	return []element.Element{element.SASLSuccess}, props, false
}
