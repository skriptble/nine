package features

import (
	"log"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

type SASL struct {
	required   bool
	retries    int
	mechanisms map[string]SASLMechanism
}

type SASLMechanism interface {
	Authenticate(el element.Element, s stream.Streamv0) (ns stream.Streamv0, success bool)
}

func NewSASL(required bool, retries int, mechs map[string]SASLMechanism) Feature {
	return &SASL{required: required, retries: retries, mechanisms: mechs}
}

func (ss *SASL) Negotiate(el element.Element, s stream.Streamv0) (ns stream.Streamv0, restart, closeStream, retry bool) {
	var success bool
	// check the el to ensure it's an auth
	if el.Tag != "auth" {
		// TODO(skriptble): Return stream error
		return s, false, true, false
	}
	// check the mechanism attribute to ensure it's one we support
	log.Println("here we go")
	mech := el.SelectAttrValue("mechanism", "")
	handler, ok := ss.mechanisms[mech]
	// if we don't support it return a stream error
	if !ok {
		// TODO(skriptble): Return stream error, unsupported mechanism
		retry = true
		ns = s
		return
	}
	// do the authentication
	s, success = handler.Authenticate(el, s)
	// if failed handle failure
	if !success {
		ss.retries--
		if ss.retries < 1 {
			// TODO(skriptble): Return stream error, too many failed auth attempts
			retry = false
		} else {
			retry = true
		}
		return s, false, false, retry
	}
	// on success send success element.
	s.WriteElement(element.SASLSuccess)

	log.Printf("%+v", s)
	return s, true, false, false
}

func (ss *SASL) TransformElement() element.Element {
	mechs := element.SASLMechanisms
	for name := range ss.mechanisms {
		log.Println("This ran!")
		mechs.Child = append(mechs.Child, element.Element{Tag: "mechanism"}.SetText(name))
	}
	return mechs
}
