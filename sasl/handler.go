package sasl

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/stream"
)

var _ stream.ElementMuxerEntry = &Handler{}

// Mechanism is the interface implemented by SASL Mechanisms.
type Mechanism interface {
	// Authenticate is the method used when data is recieved. Elements returned
	// are written directly to the stream. The modified stream properties will
	// be assigned to the stream. If challenge is true, this mechanism will be
	// used for any subsequent response elements.
	Authenticate(data string) (elems []element.Element, sc stream.StateChange, restart, challenge bool)
}

type Handler struct {
	mechs   map[string]Mechanism
	current Mechanism
	genFtr  func() (el element.Element, ok bool)
}

func NewHandler(mechs map[string]Mechanism) *Handler {
	h := new(Handler)
	h.mechs = mechs
	h.genFtr = h.negotiateFeature
	return h
}

func (h *Handler) GenerateFeature() (el element.Element, ok bool) {
	return h.genFtr()
}

func (h *Handler) negotiateFeature() (el element.Element, ok bool) {
	mechs := element.SASLMechanisms
	for name := range h.mechs {
		mechs = mechs.AddChild(element.New("mechanism").SetText(name))
	}
	return mechs, true
}

func (h *Handler) negotiateFeatureComplete() (el element.Element, ok bool) {
	return
}

func (h *Handler) HandleElement(el element.Element) (
	elems []element.Element, sc stream.StateChange, restart, close bool) {
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
		elems, sc, restart, challenge = mech.Authenticate(data)
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
		elems, sc, restart, challenge = h.current.Authenticate(data)
		if !challenge {
			h.current = nil
		}
	}
	return
}

func (h *Handler) Update(state, payload string) {
	// If we get our echo, don't send auth as a stream feature anymore.
	if state == "authenticated" {
		h.genFtr = h.negotiateFeatureComplete
	}
}

// PlainMech implements the plain SASL mechanism from RFC4616
type plainMech struct {
	auth   PlainAuthenticator
	domain string
	jid    string
}

// NewPlainMechanism creates a new SASL plain mechanism
func NewPlainMechanism(auth PlainAuthenticator, domain string) Mechanism {
	return plainMech{auth: auth, domain: domain}
}

// Authenticate implements the Mechanism interface for PlainMech
func (pm plainMech) Authenticate(data string) (elems []element.Element, sc stream.StateChange, restart, challenge bool) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return []element.Element{element.SASLFailure.MalformedRequest}, nil, false, false
	}

	res := strings.Split(string(decoded), "\000")
	if len(res) != 3 {
		return []element.Element{element.SASLFailure.MalformedRequest}, nil, false, false
	}
	identity, user, password := res[0], res[1], res[2]
	err = pm.auth.Authenticate(identity, user, password)
	// TODO: Handle different types of errors
	if err != nil {
		return []element.Element{element.SASLFailure.NotAuthorized}, nil, false, false
	}
	if identity != "" {
		user = identity
	} else {
		// TODO: Add a way to determine the address of the server for the domain
		// part of the jid (do it better than this.)
		user += "@" + pm.domain
	}

	j := jid.New(user)
	pm.jid = j.String()
	return []element.Element{element.SASLSuccess}, pm.authSuccess, true, false
}

func (pm plainMech) authSuccess() (state, payload string) {
	return "authenticated", pm.jid
}
