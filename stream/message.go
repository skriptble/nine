package stream

import "github.com/skriptble/nine/element/stanza"

type MessageMux struct {
	Tag   string
	Space string
	Type  string
	FSM   MessageHandler
}

type MessageHandler interface {
	HandleMessage(stanza.Message, Properties) ([]stanza.Stanza, Properties)
}

func (mh MessageMux) Match(m stanza.Message) bool {
	if m.Type != mh.Type {
		return false
	}

	child := m.First()
	if child.Tag != mh.Tag || child.Space != mh.Space {
		return false
	}

	return true
}
