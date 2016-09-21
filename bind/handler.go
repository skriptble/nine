package bind

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"strconv"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/stream"
)

// RouteRegister is the interface implemented by routers. This allows the bind
// handler to notify a router when a user has been bound.
type RouteRegister interface {
	RegisterRoute(jid jid.JID, s stream.Stream)
}

var _ stream.IQHandler = &Handler{}

type Handler struct {
	fg  func() (el element.Element, ok bool)
	jid string
	rr  RouteRegister
	s   stream.Stream
}

func NewHandler() *Handler {
	h := new(Handler)
	h.fg = h.negotiateFeature
	return h
}

func (h *Handler) RegisterStream(s stream.Stream) {
	h.s = s
}

func (h *Handler) AddRouteRegister(rr RouteRegister) {
	h.rr = rr
}

func (h *Handler) GenerateFeature() (element.Element, bool) {
	return h.fg()
}

func (h *Handler) negotiateFeature() (el element.Element, ok bool) {
	return element.Bind, true
}

func (h *Handler) negotiateFeatureComplete() (el element.Element, ok bool) {
	return
}

func (h *Handler) HandleIQ(iq stanza.IQ) (
	sts []stanza.Stanza, sc stream.StateChange, restart, close bool) {
	// ensure we have a bind request
	req, err := stanza.TransformBindRequest(iq)
	if err != nil {
		// TODO: Should this return an error?
		return
	}
	if req.Resource == "" {
		// TODO: Create a random resource generator
		req.Resource = genResourceID()
	}

	// Should do some resource validation here.
	// TODO: Need to use proper jids here.
	str := h.jid + "/" + req.Resource

	// TODO(skriptble): Check if the jid is nil, if it is return an error
	j := jid.New(str)
	res := stanza.NewBindResult(iq, j)
	sts = append(sts, res.TransformStanza())

	sc = stream.StateChange(func() (state, payload string) {
		return "bind", j.String()
	})
	if h.rr != nil {
		h.rr.RegisterRoute(j, h.s)
	}
	return
}

func (h *Handler) Update(state, payload string) {
	// TODO(skriptble): Add debugging traces
	switch state {
	case "authenticated":
		h.jid = payload
	case "bind":
		h.fg = h.negotiateFeatureComplete
	}
}

func genResourceID() string {
	id := make([]byte, 16)
	_, err := rand.Read(id)
	if err != nil {
		// Can't generate a UUID, generate a random int64
		return strconv.FormatInt(mrand.Int63(), 10)
	}

	id[8] = (id[8] | 0x80) & 0xBF
	id[6] = (id[6] | 0x40) & 0x4F

	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}
