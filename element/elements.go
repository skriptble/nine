package element

import "github.com/skriptble/nine/namespace"

var StartTLS = Element{Tag: "starttls", Attr: []Attr{{Key: "xmlns", Value: namespace.TLS}}}
var TLSProceed = Element{Tag: "proceed", Attr: []Attr{{Key: "xmlns", Value: namespace.TLS}}}
var StreamFeatures = Element{Space: "stream", Tag: "features"}
var SASLFailure = Element{Tag: "failure", Attr: []Attr{{Key: "xmlns", Value: namespace.SASL}}}
var SASLSuccess = Element{Tag: "success", Attr: []Attr{{Key: "xmlns", Value: namespace.SASL}}}
var SASLMechanisms = Element{Tag: "mechanisms", Attr: []Attr{{Key: "xmlns", Value: namespace.SASL}}}
var Bind = Element{Tag: "bind", Attr: []Attr{{Key: "xmlns", Value: namespace.Bind}}}
var JID = New("jid")
var Required = New("required")

var StreamError = New("stream:error")
var StreamErrBadFormat = StreamError.AddChild(New("bad-format").AddAttr("xmlns", namespace.Stream))

// SASL
var Failure = New("failure").AddAttr("xmlns", namespace.SASL)
var MalformedRequest = Failure.AddChild(New("malformed-request"))
