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

type Handler struct {
}

func NewHandler() Handler {
	return Handler{}
}

func (h Handler) HandleFeature(props stream.Properties) stream.Properties {
	if props.Status&stream.Bind != 0 || props.Status&stream.Auth == 0 {
		return props
	}

	props.Features = append(props.Features, element.Bind)
	return props
}

func (h Handler) HandleIQ(iq stanza.IQ, props stream.Properties) ([]stanza.Stanza, stream.Properties) {
	var sts []stanza.Stanza
	// ensure we have a bind request
	req, err := stanza.TransformBindRequest(iq)
	if err != nil {
		// TODO: Should this return an error?
		return sts, props
	}
	if req.Resource == "" {
		// TODO: Create a random resource generator
		req.Resource = genResourceID()
	}

	// Should do some resource validation here.
	// TODO: Need to use proper jids here.
	props.Header.To += "/" + req.Resource

	j := jid.New(props.Header.To)
	res := stanza.NewBindResult(iq, j)
	sts = append(sts, res.TransformStanza())

	props.Status = props.Status | stream.Bind
	return sts, props
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
