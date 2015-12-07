package features

import (
	"encoding/base64"
	"strings"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

type SASLPlain struct {
}

// Authenticate implements plain authentication for SASL.
func (sp SASLPlain) Authenticate(el element.Element, s stream.Streamv0) (stream.Streamv0, bool) {
	// TODO(skriptble): Implement for real to spec. Just trying to get things
	// working.
	data, err := base64.StdEncoding.DecodeString(el.Text())
	if err != nil {
		return s, false
	}

	res := strings.Split(string(data), "\000")
	if len(res) != 3 {
		return s, false
	}
	s.Header.To = res[1] + "@localhost"
	return s, true
}
