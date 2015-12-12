package stream

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

type FSMv0 interface {
	Next(Streamv0) (FSMv0, Streamv0)
}

type FSM interface {
}

type ElementHandler struct {
	Tag   string
	Space string
	FSM   ElementFSM
}

type ElementFSM interface {
	HandleElement(element.Element, Properties) ([]element.Element, Properties)
}

func (eh ElementHandler) Match(el element.Element) bool {
	if el.Space != eh.Space || el.Tag != eh.Tag {
		return false
	}
	return true
}

type MessageHandler struct {
	Tag   string
	Space string
	Type  string
	FSM   MessageFSM
}

type MessageFSM interface {
	HandleMessage(stanza.Message, Properties) ([]stanza.Stanza, Properties)
}

func (mh MessageHandler) Match(m stanza.Message) bool {
	if m.Type != mh.Type {
		return false
	}

	child := m.First()
	if child.Tag != mh.Tag || child.Space != mh.Space {
		return false
	}

	return true
}

type PresenceHandler struct {
	Tag   string
	Space string
	Type  string
	FSM   PresenceFSM
}

type PresenceFSM interface {
	HandlePresence(stanza.Presence, Properties) ([]stanza.Stanza, Properties)
}

func (ph PresenceHandler) Match(p stanza.Presence) bool {
	if p.Type != ph.Type {
		return false
	}

	child := p.First()
	if child.Tag != ph.Tag || child.Space != ph.Space {
		return false
	}

	return true
}

type IQHandler struct {
	Tag   string
	Space string
	Type  string
	FSM   IQFSM
}

type IQFSM interface {
	HandleIQ(stanza.IQ, Properties) ([]stanza.Stanza, Properties)
}

func (iqh IQHandler) Match(iq stanza.IQ) bool {
	if !match(iqh.Type, iq.Type) {
		return false
	}

	child := iq.First()
	if !match(iqh.Tag, child.Tag) || !match(iqh.Space, child.Space) {
		return false
	}

	return true
}

// match is a utility function for matching two strings. If the first string
// is a * then it is treated as a wildcard and true is returned regardless of
// the second value.
func match(a, b string) bool {
	if a == "*" {
		return true
	}

	return a == b
}
