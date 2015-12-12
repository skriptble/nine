package jid

import (
	"bytes"
	"net"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

var Empty = JID{}

// TODO: Implement XEP-0106 for escaping
var jidReplacer = strings.NewReplacer(
	`"`, "",
	"&", "",
	"'", "",
	"/", "",
	":", "",
	"<", "",
	">", "",
	"@", "",
)

type JID struct {
	local    string
	domain   string
	resource string
}

func New(str string) JID {
	// TODO(skriptble): Implement RFC7622
	var local, domain, resource string

	domain = str
	slash := strings.IndexByte(domain, '/')
	if slash != -1 {
		resource = domain[slash+1:]
		domain = domain[:slash]
	}

	at := strings.IndexByte(domain, '@')
	if at != -1 {
		local = domain[:at]
		domain = domain[at+1:]
	}

	local = parseLocal(local)
	domain = parseDomain(domain)
	resource = parseResource(resource)

	if len([]byte(local)) > 1024 || len([]byte(domain)) > 1024 || len([]byte(resource)) > 1024 {
		return Empty
	}

	return JID{
		local:    local,
		domain:   domain,
		resource: resource,
	}
}

// parseDomain parses the domain part of a jid according to RFC7622
func parseDomain(domain string) string {
	ip := net.ParseIP(domain)
	if ip != nil {
		return ip.String()
	}

	domain, err := idna.ToUnicode(domain)
	if err != nil {
		return ""
	}
	return domain
}

// parseLocal parses the local part of a jid according to RFC7622
func parseLocal(local string) string {
	local = width.Fold.String(local)
	b := bytes.ToLower([]byte(local))
	local = string(b)
	local = norm.NFC.String(local)
	return jidReplacer.Replace(local)
}

// parseResource parses the resource part of a jid according to RFC7622
func parseResource(resource string) string {
	return norm.NFC.String(resource)
}

func (j JID) Local() string {
	return j.local
}

func (j JID) Domain() string {
	return j.domain
}

func (j JID) SetDomain(domain string) JID {
	j.domain = parseDomain(domain)
	return j
}

func (j JID) Resource() string {
	return j.resource
}

func (j JID) String() string {
	res := ""
	if j.local != "" {
		res += j.local + "@"
	}
	res += j.domain
	if j.resource != "" {
		res += "/" + j.resource
	}

	return res
}
