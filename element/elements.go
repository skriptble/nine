package element

import "github.com/skriptble/nine/namespace"

// TLS
var StartTLS = New("starttls").AddAttr("xmlns", namespace.TLS)
var StartTLSRequired = StartTLS.AddChild(New("required"))
var TLSProceed = New("proceed").AddAttr("xmlns", namespace.TLS)
var TLSFailure = New("failure").AddAttr("xmlns", namespace.TLS)

// Bind
var Bind = Element{Tag: "bind", Attr: []Attr{{Key: "xmlns", Value: namespace.Bind}}}

// TODO: Move this to Ten
var Session = New("session").AddAttr("xmlns", namespace.Session)
var JID = New("jid")
var Required = New("required")

// Stream
var StreamFeatures = New("stream:features")
var se = New("stream:error")
var StreamError = struct {
	Base, BadFormat, BadNamespacePrefix, Conflict, ConnectionTimeout, HostGone, HostUnknown,
	ImproperAddressing, InternalServerError, InvalidFrom, InvalidNamespace, InvalidXML, NotAuthorized,
	NotWellFormed, PolicyViolation, RemoteConnectionFailed, Reset, ResourceConstraint, RestrictedXML,
	SeeOtherHost, SystemShutdown, UndefinedCondition, UnsupportedEncoding, UnsupportedFeature,
	UnsupportedStanzaType, UnsupportedVersion Element
}{
	Base:                   se,
	BadFormat:              se.AddChild(streamErr("bad-format")),
	BadNamespacePrefix:     se.AddChild(streamErr("bad-namespace-preix")),
	Conflict:               se.AddChild(streamErr("conflict")),
	ConnectionTimeout:      se.AddChild(streamErr("connection-timeout")),
	HostGone:               se.AddChild(streamErr("host-gone")),
	HostUnknown:            se.AddChild(streamErr("host-unknown")),
	ImproperAddressing:     se.AddChild(streamErr("improper-addressing")),
	InternalServerError:    se.AddChild(streamErr("internal-server-error")),
	InvalidFrom:            se.AddChild(streamErr("invalid-from")),
	InvalidNamespace:       se.AddChild(streamErr("invalid-namespace")),
	InvalidXML:             se.AddChild(streamErr("invalid-xml")),
	NotAuthorized:          se.AddChild(streamErr("not-authorized")),
	NotWellFormed:          se.AddChild(streamErr("not-well-formed")),
	PolicyViolation:        se.AddChild(streamErr("policy-violation")),
	RemoteConnectionFailed: se.AddChild(streamErr("remote-connection-closed")),
	Reset:                 se.AddChild(streamErr("reset")),
	ResourceConstraint:    se.AddChild(streamErr("resource-constraint")),
	RestrictedXML:         se.AddChild(streamErr("restricted-xml")),
	SeeOtherHost:          se.AddChild(streamErr("see-other-host")),
	SystemShutdown:        se.AddChild(streamErr("system-shutdown")),
	UndefinedCondition:    se.AddChild(streamErr("undefined-condition")),
	UnsupportedEncoding:   se.AddChild(streamErr("unsupported-encoding")),
	UnsupportedFeature:    se.AddChild(streamErr("unsupported-feature")),
	UnsupportedStanzaType: se.AddChild(streamErr("unsupported-stanza-type")),
	UnsupportedVersion:    se.AddChild(streamErr("unsupported-version")),
}
var StreamErrorBase = New("stream:error")
var StreamErrBadFormat = StreamErrorBase.AddChild(New("bad-format").AddAttr("xmlns", namespace.Stream))

// SASL
var SASL = struct {
	Abort, Failure, Mechanisms, Success Element
}{
	Abort:      New("abort").AddAttr("xmlns", namespace.SASL),
	Failure:    New("failure").AddAttr("xmlns", namespace.SASL),
	Mechanisms: New("mechanisms").AddAttr("xmlns", namespace.SASL),
	Success:    New("success").AddAttr("xmlns", namespace.SASL),
}
var SASLFailure = struct {
	Aborted, AccountDisabled, CredentialsExpired, EncryptionRequired, IncorrectEncoding,
	InvalidAuthzid, InvalidMechanism, MalformedRequest, MechanismTooWeak, NotAuthorized,
	TemporaryAuthFailure Element
}{
	Aborted:              SASL.Failure.AddChild(saslErr("aborted")),
	AccountDisabled:      SASL.Failure.AddChild(saslErr("account-disabled")),
	CredentialsExpired:   SASL.Failure.AddChild(saslErr("credentials-expired")),
	EncryptionRequired:   SASL.Failure.AddChild(saslErr("encryption-required")),
	IncorrectEncoding:    SASL.Failure.AddChild(saslErr("incorrect-encoding")),
	InvalidAuthzid:       SASL.Failure.AddChild(saslErr("invalid-authzid")),
	InvalidMechanism:     SASL.Failure.AddChild(saslErr("invalid-mechanism")),
	MalformedRequest:     SASL.Failure.AddChild(saslErr("malformed-request")),
	MechanismTooWeak:     SASL.Failure.AddChild(saslErr("mechanism-too-weak")),
	NotAuthorized:        SASL.Failure.AddChild(saslErr("not-authorized")),
	TemporaryAuthFailure: SASL.Failure.AddChild(saslErr("temporary-auth-failure")),
}
var SASLSuccess = Element{Tag: "success", Attr: []Attr{{Key: "xmlns", Value: namespace.SASL}}}
var SASLMechanisms = Element{Tag: "mechanisms", Attr: []Attr{{Key: "xmlns", Value: namespace.SASL}}}

// Stanza
// TODO: These should be implemented as Stanzas, not Elements.
var Stanza = struct {
	BadRequest, Conflict, FeatureNotImplemented, Forbidden, Gone, InternalServerError,
	ItemNotFound, JidMalformed, NotAcceptable, NotAllowed, NotAuthorized, PolicyViolation,
	RecipientUnavailable, Redirect, RegistrationRequired, RemoteServerNotFound, RemoteServerTimeout,
	ResourceConstraint, ServiceUnavailable, SubscriptionRequired, UndefinedCondition,
	UnexpectedRequest Element
}{
	BadRequest:            stanzaErrType("modify").AddChild(stanzaErr("bad-request")),
	Conflict:              stanzaErrType("cancel").AddChild(stanzaErr("conflict")),
	FeatureNotImplemented: stanzaErrType("cancel").AddChild(stanzaErr("feature-not-implemented")),
	Forbidden:             stanzaErrType("auth").AddChild(stanzaErr("forbidden")),
	Gone:                  stanzaErrType("cancel").AddChild(stanzaErr("gone")),
	InternalServerError:   stanzaErrType("cancel").AddChild(stanzaErr("internal-server-error")),
	ItemNotFound:          stanzaErrType("cancel").AddChild(stanzaErr("item-not-found")),
	JidMalformed:          stanzaErrType("modify").AddChild(stanzaErr("jid-malformed")),
	NotAcceptable:         stanzaErrType("modify").AddChild(stanzaErr("not-acceptable")),
	NotAllowed:            stanzaErrType("cancel").AddChild(stanzaErr("not-allowed")),
	NotAuthorized:         stanzaErrType("auth").AddChild(stanzaErr("not-authorized")),
	PolicyViolation:       stanzaErrType("modify").AddChild(stanzaErr("policy-violation")),
	RecipientUnavailable:  stanzaErrType("wait").AddChild(stanzaErr("recipient-unavailable")),
	Redirect:              stanzaErrType("modify").AddChild(stanzaErr("redirect")),
	RegistrationRequired:  stanzaErrType("auth").AddChild(stanzaErr("registration-required")),
	RemoteServerNotFound:  stanzaErrType("cancel").AddChild(stanzaErr("remote-server-not-found")),
	RemoteServerTimeout:   stanzaErrType("wait").AddChild(stanzaErr("remote-server-timeout")),
	ResourceConstraint:    stanzaErrType("wait").AddChild(stanzaErr("resource-constraint")),
	ServiceUnavailable:    stanzaErrType("cancel").AddChild(stanzaErr("service-unavailable")),
	SubscriptionRequired:  stanzaErrType("auth").AddChild(stanzaErr("subscription-required")),
	UndefinedCondition:    stanzaErrType("modify").AddChild(stanzaErr("undefined-condition")),
	UnexpectedRequest:     stanzaErrType("modify").AddChild(stanzaErr("unexpected-request")),
}

func streamErr(tag string) Element {
	return New(tag).AddAttr("xmlns", namespace.Stream)
}

func saslErr(tag string) Element {
	return New(tag)
}

func stanzaErr(tag string) Element {
	return New(tag).AddAttr("xmlns", namespace.Stanza)
}

func stanzaErrType(name string) Element {
	return New("error").AddAttr("type", name)
}
