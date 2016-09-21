package bind

import (
	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/stream"
)

var _ stream.IQHandler = &SessionHandler{}

// SessionHandler belongs in ten but is a placeholder here for now to get
// clients to connect and hold a connection.
type SessionHandler struct {
	fg func() (el element.Element, ok bool)
}

func NewSessionHandler() *SessionHandler {
	sh := new(SessionHandler)
	sh.fg = sh.negotiateFeature
	return sh
}

func (sh *SessionHandler) GenerateFeature() (el element.Element, ok bool) {
	return sh.fg()
}

func (sh *SessionHandler) negotiateFeature() (el element.Element, ok bool) {
	return element.Session, true
}

func (sh *SessionHandler) negotiateFeatureComplete() (el element.Element, ok bool) {
	return
}

func (sh *SessionHandler) HandleIQ(iq stanza.IQ) (
	sts []stanza.Stanza, sc stream.StateChange, restart, close bool) {
	to := jid.New(iq.From)
	from := jid.New(iq.To)
	res := stanza.NewIQResult(to, from, iq.ID, stanza.IQResult)
	sts = append(sts, res.TransformStanza())
	sc = stream.StateChange(func() (state, payload string) {
		return "session-established", ""
	})
	return
}

func (sh *SessionHandler) Update(state, payload string) {
	if state == "session-established" {
		sh.fg = sh.negotiateFeatureComplete
	}
}
