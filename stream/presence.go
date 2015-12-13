package stream

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
)

type PresenceMux struct {
	Tag   string
	Space string
	Type  string
	FSM   PresenceHandler

	handlers []pEntry
	err      error
}

type pEntry struct {
	space, tag string
	sType      string
	h          PresenceHandler
}

func NewPresenceMux() PresenceMux {
	return PresenceMux{}
}

func (pm PresenceMux) HandleElement(el element.Element, p Properties) ([]element.Element, Properties) {
	return []element.Element{}, p
}

type PresenceHandler interface {
	HandlePresence(stanza.Presence, Properties) ([]stanza.Stanza, Properties)
}

func (ph PresenceMux) Match(p stanza.Presence) bool {
	if p.Type != ph.Type {
		return false
	}

	child := p.First()
	if child.Tag != ph.Tag || child.Space != ph.Space {
		return false
	}

	return true
}
