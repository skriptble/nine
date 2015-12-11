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
	HandleElement(element.Element, Properties) ([]element.Element, ElementFSM, Properties)
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
	HandleMessage(stanza.Message, Properties) ([]stanza.Stanza, MessageFSM, Properties)
}

type PresenceHandler struct {
	Tag   string
	Space string
	Type  string
	FSM   PresenceFSM
}

type PresenceFSM interface {
	HandlePresence(stanza.Presence, Properties) ([]stanza.Stanza, PresenceFSM, Properties)
}

type IQHandler struct {
	Tag   string
	Space string
	Type  string
	FSM   IQFSM
}

type IQFSM interface {
	HandleIQ(stanza.IQ, Properties) ([]stanza.Stanza, IQFSM, Properties)
}

func (iqh IQHandler) Match(iq stanza.IQ) bool {
	if iq.Type != iqh.Type {
		return false
	}

	child := iq.First()
	if child.Tag != iqh.Tag || child.Space != iqh.Space {
		return false
	}

	return true
}
