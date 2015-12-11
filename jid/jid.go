package jid

import "strings"

type JID struct {
	local    string
	domain   string
	resource string
}

func NewJID(str string) JID {
	// TODO(skriptble): Implement RFC7622
	var local, domain, resource string

	domain = str
	slash := strings.IndexByte(domain, '/')
	if slash != -1 {
		resource = domain[slash+1:]
		domain = domain[slash:]
	}

	at := strings.IndexByte(domain, '@')
	if at != -1 {
		local = domain[:at]
		domain = domain[at+1:]
	}

	return JID{
		local:    local,
		domain:   domain,
		resource: resource,
	}
}

func (j JID) Local() string {
	return j.local
}

func (j JID) Domain() string {
	return j.domain
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
