package transport

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/jid"
	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/stream"
)

func TestWriteElement(t *testing.T) {
	t.Parallel()

	var want, got []byte

	el := element.New("testing").AddAttr("xmlns", "foo:bar")
	want = el.WriteBytes()
	got = make([]byte, len(want))

	read, write := net.Pipe()
	tcpTsp := NewTCP(write, stream.Receiving, nil, false)

	go func() {
		_, err := read.Read(got)
		if err != nil {
			t.Errorf("Received error while reading from connection: %s", err)
		}
	}()

	err := tcpTsp.WriteElement(el)
	if err != nil {
		t.Errorf("Unexpected error from WriteElement: %s", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to write element to TCP stream.")
		t.Errorf("\nWant:%v\nGot :%v", want, got)
	}
}

func TestWriteElementError(t *testing.T) {
	t.Parallel()

	var want, got error

	want = io.ErrClosedPipe

	el := element.New("testing")
	_, pipe := net.Pipe()
	tcpTsp := NewTCP(pipe, stream.Receiving, nil, false)
	err := pipe.Close()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	got = tcpTsp.WriteElement(el)
	if got != want {
		t.Error("Should receive error from connection when writing element.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestWriteStanza(t *testing.T) {
	t.Parallel()

	var want, got []byte

	to, from := jid.New("foo@bar"), jid.New("baz@quux")
	iq := stanza.NewIQResult(to, from, "test", stanza.IQResult)
	want = iq.TransformElement().WriteBytes()
	got = make([]byte, len(want))

	read, write := net.Pipe()
	tcpTsp := NewTCP(write, stream.Receiving, nil, false)

	go func() {
		_, err := read.Read(got)
		if err != nil {
			t.Errorf("Received error while reading from connection: %s", err)
		}
	}()

	err := tcpTsp.WriteStanza(iq.TransformStanza())
	if err != nil {
		t.Errorf("Unexpected error from WriteElement: %s", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to write element to TCP stream.")
		t.Errorf("\nWant:%v\nGot :%v", want, got)
	}
}

func TestWriteStanzaError(t *testing.T) {
	t.Parallel()

	var want, got error

	want = io.ErrClosedPipe

	st := stanza.Stanza{}
	_, pipe := net.Pipe()
	tcpTsp := NewTCP(pipe, stream.Receiving, nil, false)
	err := pipe.Close()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	got = tcpTsp.WriteStanza(st)
	if got != want {
		t.Error("Should receive error from connection when writing element.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestNext(t *testing.T) {
	t.Parallel()

	var want, got interface{}
	var err error
	var el element.Element
	pipe1, pipe2 := net.Pipe()
	el = element.New("testing").AddAttr("foo", "bar").
		SetText("random text").
		AddChild(element.New("baz-quux"))
	tcpTsp := NewTCP(pipe1, stream.Receiving, nil, true)

	// Should be able to get a token from the transport
	go func() {
		_, err := pipe2.Write(el.WriteBytes())
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
	}()

	want = el
	got, err = tcpTsp.Next()
	if err != nil {
		t.Errorf("An unexpected error occurred: %s", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Error("Should be able to get a token from the transport.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Stream element should return token and not attempt to read the entire stream.
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, nil, true)
	go func() {
		_, err := pipe2.Write(stream.Header{}.WriteBytes())
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
		_, err = pipe2.Write([]byte("<foo/></stream:stream>"))
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
	}()
	el, err = tcpTsp.Next()
	if err != nil {
		t.Errorf("An unexpected error occurred: %s", err)
	}
	if el.Space != namespace.Stream || el.Tag != "stream" {
		t.Error("Stream element should return token and not attempt to read the entire stream.")
	}
	got, err = tcpTsp.Next()
	if err != nil {
		t.Errorf("An unexpected error occurred: %s", err)
	}
	want = element.New("foo")
	if !reflect.DeepEqual(want, got) {
		t.Error("Stream element should return token and not attempt to read the entire stream.")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestStartTLS(t *testing.T) {
	t.Parallel()

	// Should hijack starttls elements and perform a tls upgrade
	var certpool = x509.NewCertPool()
	if !certpool.AppendCertsFromPEM(caCertificatePEM) {
		t.Errorf("Could not append CA Certificate to pool.")
	}

	cert, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		t.Errorf("Unexpected error while loading key pairs: %s", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certpool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   "testing-cert",
	}

	serverPipe, clientPipe := net.Pipe()
	tcpTsp := NewTCP(serverPipe, stream.Receiving, tlsConfig, true)

	tlsConfig = &tls.Config{
		RootCAs:    certpool,
		ServerName: "testing-cert",
	}
	client := tls.Client(clientPipe, tlsConfig)

	go func() {
		_, err := clientPipe.Write(element.StartTLS.WriteBytes())
		if err != nil {
			t.Errorf("Error while writing startTLS element: %s", err)
		}
		elemBytes := element.TLSProceed.WriteBytes()
		proceed := make([]byte, len(elemBytes))
		_, err = clientPipe.Read(proceed)
		if err != nil {
			t.Errorf("Error while reading clientPipe: %s", err)
		}
		if !bytes.Equal(elemBytes, proceed) {
			t.Error("Should receive proceed element.")
			t.Errorf("\nWant:%s\nGot :%s", elemBytes, proceed)
		}
		err = client.Handshake()
		if err != nil {
			t.Errorf("Error while performing handshake: %s", err)
		}
	}()

	el, got := tcpTsp.Next()
	if !reflect.DeepEqual(element.Element{}, el) {
		t.Errorf("Expected empty element, received %+v", el)
	}
	if got != stream.ErrRequireRestart {
		t.Errorf("Expected require restart error, received %s", err)
	}

	// Should return error from Handshake
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certpool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   "testing-cert",
	}

	serverPipe, clientPipe = net.Pipe()
	tcpTsp = NewTCP(serverPipe, stream.Receiving, tlsConfig, true)

	tlsConfig = &tls.Config{
		ServerName: "testing-cert",
	}
	client = tls.Client(clientPipe, tlsConfig)

	go func() {
		_, err := clientPipe.Write(element.StartTLS.WriteBytes())
		if err != nil {
			t.Errorf("Error while writing startTLS element: %s", err)
		}
		elemBytes := element.TLSProceed.WriteBytes()
		proceed := make([]byte, len(elemBytes))
		_, err = clientPipe.Read(proceed)
		if err != nil {
			t.Errorf("Error while reading clientPipe: %s", err)
		}
		if !bytes.Equal(elemBytes, proceed) {
			t.Error("Should receive proceed element.")
			t.Errorf("\nWant:%s\nGot :%s", elemBytes, proceed)
		}
		err = clientPipe.Close()
		if err != nil {
			t.Errorf("Error while closing clientPipe: %s", err)
		}
	}()
	el, got = tcpTsp.Next()
	if !reflect.DeepEqual(element.Element{}, el) {
		t.Errorf("Expected empty element, received %+v", el)
	}
	if got != io.EOF {
		t.Errorf("Expected io.EOF, received %s", got)
	}

	// Should return error from writing element.
	serverPipe, clientPipe = net.Pipe()
	tcpTsp = NewTCP(serverPipe, stream.Receiving, tlsConfig, true)

	go func() {
		_, err := clientPipe.Write(element.StartTLS.WriteBytes())
		if err != nil {
			t.Errorf("Error while writing startTLS element: %s", err)
		}
		err = clientPipe.Close()
		if err != nil {
			t.Errorf("Error while closing clientPipe: %s", err)
		}
	}()
	el, got = tcpTsp.Next()
	if !reflect.DeepEqual(element.Element{}, el) {
		t.Errorf("Expected empty element, received %+v", el)
	}
	if got != io.ErrClosedPipe {
		t.Errorf("Expected %s, received %s", io.ErrClosedPipe, got)
	}
}

func TestNextError(t *testing.T) {
	t.Parallel()

	var want, got error
	pipe1, pipe2 := net.Pipe()
	tcpTsp := NewTCP(pipe1, stream.Receiving, nil, true)

	// Error from xml.Decoder should be returned
	go func() {
		_, err := pipe2.Write([]byte("</whoops>"))
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
	}()
	_, got = tcpTsp.Next()
	if _, ok := got.(*xml.SyntaxError); !ok {
		t.Error("Error from xml.Decoder should be returned.")
		t.Errorf("Wanted xml.SyntaxError, Got:(%T)%s", got, got)
	}

	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, nil, true)
	go func() {
		_, err := pipe2.Write([]byte("<foo><bar></whoops>"))
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
	}()
	_, got = tcpTsp.Next()
	if _, ok := got.(*xml.SyntaxError); !ok {
		t.Error("Error from xml.Decoder should be returned.")
		t.Errorf("Wanted xml.SyntaxError, Got:(%T)%s", got, got)
	}

	// Receiving an xml end element should return stream.ErrStreamClosed
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, nil, true)
	go func() {
		_, err := pipe2.Write(stream.Header{}.WriteBytes())
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
		_, err = pipe2.Write([]byte("</stream:stream>"))
		if err != nil {
			t.Errorf("An unexpected error occurred: %s", err)
		}
	}()
	want = stream.ErrStreamClosed
	_, err := tcpTsp.Next()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	_, got = tcpTsp.Next()
	if !reflect.DeepEqual(want, got) {
		t.Error("Receiving an xml end element should return stream.ErrStreamClosed.")
		t.Errorf("\nWant:%s\nGot :%s", got, got)
	}
}

func TestStartInitiating(t *testing.T) {
	t.Parallel()

	var want, got []byte
	var props stream.Properties
	var err error

	pipe1, pipe2 := net.Pipe()
	tcpTsp := NewTCP(pipe1, stream.Initiating, nil, false)
	props.Header = stream.Header{}

	// Should get error when starting stream with empty header
	_, err = tcpTsp.Start(props)
	if err != stream.ErrHeaderNotSet {
		t.Error("Should get error when starting stream with empty header.")
		t.Errorf("\nWant:%s\nGot :%s", stream.ErrHeaderNotSet, err)
	}

	// Should write header to underlying connection
	hdr := stream.Header{To: "foo", From: "bar"}
	want = hdr.WriteBytes()
	props.Header = hdr
	got = make([]byte, len(want))
	go func() {
		_, err := pipe2.Read(got)
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
	}()
	_, err = tcpTsp.Start(props)
	if err != nil {
		t.Errorf("Unexpected error while starting stream: %s", err)
	}
	if !bytes.Equal(want, got) {
		t.Error("Should write header to underlying connection")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestStartReceiving(t *testing.T) {
	t.Parallel()

	var want, got []byte
	var props, gotProps stream.Properties
	var err, wantErr error

	pipe1, pipe2 := net.Pipe()
	tcpTsp := NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	props.Header = stream.Header{}

	// Should return Domain Not Set error if the domain isnot set on the
	// stream properties.
	_, err = tcpTsp.Start(props)
	if err != stream.ErrDomainNotSet {
		t.Error("Should return ErrDomainNotSet error if the domain is no set on the properties.")
		t.Errorf("\nWant:%s\nGot :%s", stream.ErrDomainNotSet, err)
	}

	// Should return error from Next
	props.Domain = "localhost"
	err = pipe2.Close()
	if err != nil {
		t.Errorf("Unexpected error from pipe2.Close: %s", err)
	}
	_, err = tcpTsp.Start(props)
	if err != io.EOF {
		t.Error("Should return error from Next")
		t.Errorf("\nWant:%s\nGot :%s", io.EOF, err)
	}

	// Should return error from NewHeader
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	go func() {
		_, err := pipe2.Write([]byte("<baz xmlns='foo:bar'/>"))
		if err != nil {
			t.Errorf("Unexpected error while writing to pipe2: %s", err)
		}
	}()
	_, err = tcpTsp.Start(props)
	wantErr = fmt.Errorf("Element is not <stream:stream> it is a <foo:bar:baz>")
	if err.Error() != wantErr.Error() {
		t.Error("Should return error from NewHeader")
		t.Errorf("\nWant:%s\nGot :%s", wantErr, err)
	}

	// Should send HostUnknown if the to field of the header does not match the
	// Domain field on properties
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	go func() {
		hdr := stream.Header{To: "not-localhost", From: "foo@bar"}
		_, err := pipe2.Write(hdr.WriteBytes())
		if err != nil {
			t.Errorf("Unexpected error while writing to pipe2: %s", err)
		}
		hdr.To, hdr.From = hdr.From, "localhost"
		want = hdr.WriteBytes()
		// We need to add an extra 36 bytes for the id length
		hdrLen := len(want) + 36
		want = append(want, element.StreamError.HostUnknown.WriteBytes()...)
		// We need to add an extra 36 bytes for the id length
		got = make([]byte, len(want)+36)
		_, err = pipe2.Read(got)
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
		_, err = pipe2.Read(got[hdrLen:])
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
	}()
	gotProps, err = tcpTsp.Start(props)
	if err != nil {
		t.Errorf("Unexpected error from Start: %s", err)
	}
	if gotProps.Status != stream.Closed {
		t.Error("Expected stream to be marked as closed after host unknown error")
	}
	// Need to remove stream ID before comparing
	idx := bytes.Index(got, []byte("id='"))
	if idx == -1 {
		t.Error("Received stream is missing id attribute")
	}
	// We slice the id out of the received stream header.
	got = append(got[:idx+4], got[idx+40:]...)
	if !bytes.Equal(want, got) {
		t.Error("Should send HostUnknown if the to field of the header does not match the domain field on properties.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}

	// Should return error from writing header to the underlying connection
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	go func() {
		hdr := stream.Header{To: "localhost", From: "foo@bar"}
		_, err := pipe2.Write(hdr.WriteBytes())
		if err != nil {
			t.Errorf("Unexpected error while writing to pipe2: %s", err)
		}
		err = pipe2.Close()
		if err != nil {
			t.Errorf("Unexpected error while closing pipe2: %s", err)
		}
	}()
	gotProps, err = tcpTsp.Start(props)
	if err != io.ErrClosedPipe {
		t.Error("Should return error from writing header to the underlying connection.")
		t.Errorf("\nWant:%s\nGot :%s", io.ErrClosedPipe, err)
	}

	// Should set the To field to the properties To field if it is set
	props.To = "authenticatedFoo@bar"

	// Should overwrite stream features if there is a tls config and the stream
	// is not yet secure
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	go func() {
		hdr := stream.Header{To: "localhost", From: "foo@bar"}
		_, err := pipe2.Write(hdr.WriteBytes())
		if err != nil {
			t.Errorf("Unexpected error while writing to pipe2: %s", err)
		}
		// Doing two tests at the same time, because they are orthogonal
		hdr.To, hdr.From = "authenticatedFoo@bar", "localhost"
		want = hdr.WriteBytes()
		// We need to add an extra 36 bytes for the id length
		hdrLen := len(want) + 36
		ftrs := element.StreamFeatures.AddChild(
			element.StartTLS.AddChild(
				element.Required),
		)
		want = append(want, ftrs.WriteBytes()...)
		// We need to add an extra 36 bytes for the id length
		got = make([]byte, len(want)+36)
		_, err = pipe2.Read(got)
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
		_, err = pipe2.Read(got[hdrLen:])
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
	}()
	gotProps, err = tcpTsp.Start(props)
	if err != nil {
		t.Errorf("Unexpected error from Start: %s", err)
	}
	// Need to remove stream ID before comparing
	idx = bytes.Index(got, []byte("id='"))
	if idx == -1 {
		t.Error("Received stream is missing id attribute")
	}
	// We slice the id out of the received stream header.
	got = append(got[:idx+4], got[idx+40:]...)
	if !bytes.Equal(want, got) {
		t.Error("Should overwrite stream features if there is a tls config and the stream is not yet secure.")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}

	// Should be able to start stream
	props.Features = append(props.Features, element.Bind)
	pipe1, pipe2 = net.Pipe()
	tcpTsp = NewTCP(pipe1, stream.Receiving, &tls.Config{}, true)
	tcpTsp.(*TCP).secure = true
	go func() {
		hdr := stream.Header{To: "localhost", From: "foo@bar"}
		_, err := pipe2.Write(hdr.WriteBytes())
		if err != nil {
			t.Errorf("Unexpected error while writing to pipe2: %s", err)
		}
		// Doing two tests at the same time, because they are orthogonal
		hdr.To, hdr.From = "authenticatedFoo@bar", "localhost"
		want = hdr.WriteBytes()
		// We need to add an extra 36 bytes for the id length
		hdrLen := len(want) + 36
		ftrs := element.StreamFeatures.AddChild(element.Bind)
		want = append(want, ftrs.WriteBytes()...)
		// We need to add an extra 36 bytes for the id length
		got = make([]byte, len(want)+36)
		_, err = pipe2.Read(got)
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
		_, err = pipe2.Read(got[hdrLen:])
		if err != nil {
			t.Errorf("Unexpected error while reading from pipe2: %s", err)
		}
	}()
	gotProps, err = tcpTsp.Start(props)
	if err != nil {
		t.Errorf("Unexpected error from Start: %s", err)
	}
	// Need to remove stream ID before comparing
	idx = bytes.Index(got, []byte("id='"))
	if idx == -1 {
		t.Error("Received stream is missing id attribute")
	}
	// We slice the id out of the received stream header.
	got = append(got[:idx+4], got[idx+40:]...)
	if !bytes.Equal(want, got) {
		t.Error("Should be able to start stream")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

var caCertificatePEM = []byte(`
-----BEGIN CERTIFICATE-----
MIIDNTCCAh2gAwIBAgIJAIthtvmND9xUMA0GCSqGSIb3DQEBCwUAMBYxFDASBgNV
BAMMC0Vhc3ktUlNBIENBMB4XDTE1MTEyNzE3MzQ0OVoXDTI1MTEyNDE3MzQ0OVow
FjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDkx0RO22jY31JLRbIMAo9ODffgKi/YPumogVklM8C60AlRsn64opgy
bWUW/K4bIlVAiQ3hQJvgUgnWPEHjm+VEQ2PxsxWKdYKVG7KVTmXT9feX3D6nyS+Q
QIJnfv4hFFkM/TtzfWNpREuKKhovUfk7oC8baPMuuAg1HgrdvGkJobOSjJzngFU5
ks9Hm40hhC9aZ2WRy2xVEw31dYSwwkU9DwUgVbwT/8eAXFaNQ//6Xo9C2YsYmxwk
EdYuIlaBQc7IpmELHCVq/wpotHYah0Y6R+2j75KUsr30cHya7GG6kYtrJTkIMFjj
uPj9R00vVj7xFVd4TsBgObEyUwjcCiyPAgMBAAGjgYUwgYIwHQYDVR0OBBYEFBlB
+8tm5v5TvG4wJybQEqzyY8YgMEYGA1UdIwQ/MD2AFBlB+8tm5v5TvG4wJybQEqzy
Y8YgoRqkGDAWMRQwEgYDVQQDDAtFYXN5LVJTQSBDQYIJAIthtvmND9xUMAwGA1Ud
EwQFMAMBAf8wCwYDVR0PBAQDAgEGMA0GCSqGSIb3DQEBCwUAA4IBAQAJMkZ+SaeD
U+Pe/QfkPiri9EMFaWnQM2OYQCeNAc0rCcOaPYLWvYZs3YUBFKSrikVnBLe1kArD
NsRPZaX3hxHst3GUiy6ZOpepVz0wJNmaYCFXpIx6Y+kR1jjiM6y4bg81C9O1SORl
/9xTzIcqGCkVur10sHkuKNVuMmW4dEByGJZOa8qb3yexBsQMY0aJxvjmyIriNRHZ
CXxIM9A3V0Z1G251p1PKYcTIGMGLr/mSxgUL8TxhzyX9faZVo5SxnTzyAzP1eBPr
iZ48V65NZuqzyVm5lw9qdzmAP5SR50OuOcD9vtbGbmk36jUT7q9wwNqiq1gjktIp
PBjmGRpDvT7A
-----END CERTIFICATE-----
`)

var certificatePEM = []byte(`
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: 4 (0x4)
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: CN=Easy-RSA CA
        Validity
            Not Before: Dec 18 09:42:58 2015 GMT
            Not After : Dec 15 09:42:58 2025 GMT
        Subject: CN=testing-cert
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
                Modulus:
                    00:c1:60:0a:24:86:3f:90:ed:23:11:0a:33:b1:3d:
                    99:be:08:49:42:63:37:54:ce:b0:27:6c:72:15:94:
                    96:95:e7:1f:4e:b1:64:11:9e:c7:80:3e:38:db:31:
                    9f:1a:75:d3:a7:d9:95:73:9f:89:f9:e8:34:68:7f:
                    5f:7b:40:57:ba:70:ef:31:a6:32:76:34:cb:31:dd:
                    2d:a8:27:45:6b:f4:db:b1:10:ed:26:6b:86:85:aa:
                    cf:65:b2:a3:ff:5b:2c:20:bf:5c:01:4b:85:fa:a1:
                    bf:8a:78:10:9e:89:b3:39:76:8f:72:9a:36:ef:3b:
                    75:f1:4b:72:63:8b:19:30:cc:d7:02:51:b9:ee:9e:
                    23:09:5d:0e:94:91:d2:0f:d0:9e:29:b4:e4:22:c7:
                    f5:e8:04:f0:b4:4d:2e:fb:12:64:3e:37:ab:12:ca:
                    a7:c2:fd:45:32:41:4d:26:35:55:6a:db:38:bf:08:
                    9a:a2:ac:48:34:c4:d8:2a:72:94:ff:31:07:c0:20:
                    df:29:1a:74:cc:a6:31:45:5a:d8:8f:96:14:bb:bd:
                    1a:e5:18:17:3f:41:57:95:78:1f:2c:c8:44:25:cd:
                    50:64:87:30:a3:fa:bf:bf:cf:27:4d:85:c0:4f:12:
                    8f:fd:c1:ec:33:73:bc:25:42:bb:01:40:cc:60:d6:
                    40:ef
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Basic Constraints:
                CA:FALSE
            X509v3 Subject Key Identifier:
                B7:3F:EC:A0:46:B7:2C:26:27:69:AB:4E:E6:AB:7D:E7:86:09:6B:15
            X509v3 Authority Key Identifier:
                keyid:19:41:FB:CB:66:E6:FE:53:BC:6E:30:27:26:D0:12:AC:F2:63:C6:20
                DirName:/CN=Easy-RSA CA
                serial:8B:61:B6:F9:8D:0F:DC:54

            X509v3 Extended Key Usage:
                TLS Web Server Authentication
            X509v3 Key Usage:
                Digital Signature, Key Encipherment
    Signature Algorithm: sha256WithRSAEncryption
         3c:54:82:8c:e2:4d:88:69:d5:97:b0:5b:7d:1d:98:2f:58:41:
         0f:55:c0:e8:03:f7:cb:6f:79:13:19:9c:8b:b7:80:38:0c:fd:
         fe:31:ec:95:7d:27:52:51:31:af:fd:7b:77:a6:41:b8:7d:25:
         f5:a7:bd:b7:4b:5d:92:99:2e:96:a4:47:3c:b0:09:7b:99:b5:
         6b:b9:a9:b7:67:a3:b1:52:87:aa:57:81:fa:ed:e9:aa:70:f3:
         eb:d9:67:e7:1c:5d:61:fa:81:a1:fd:12:f2:56:c6:2e:df:d5:
         b3:8b:78:7d:30:77:fb:6c:26:03:f9:1c:2a:14:69:71:df:9e:
         dd:c1:c3:8f:d3:d3:ef:12:89:57:bb:30:cb:ae:98:89:71:da:
         93:3d:4e:ea:1b:c5:ad:71:b4:a3:5d:c9:9e:b2:57:e1:5c:39:
         d2:53:91:2f:e0:e1:a1:d6:f0:3e:ff:cc:0c:eb:42:d3:b1:c3:
         35:cf:6f:52:90:55:6d:c2:0b:60:70:3c:d5:5c:12:7a:51:3d:
         b9:e9:82:86:2b:08:88:23:9d:90:0f:85:fb:bb:b4:ca:b6:38:
         ee:3f:59:36:fd:79:96:aa:0f:2d:37:3d:ed:1b:1d:e9:c7:cb:
         97:99:c5:90:1c:05:d4:90:17:17:9f:b5:2c:1d:d8:13:97:07:
         8c:3f:51:8d
-----BEGIN CERTIFICATE-----
MIIDQDCCAiigAwIBAgIBBDANBgkqhkiG9w0BAQsFADAWMRQwEgYDVQQDDAtFYXN5
LVJTQSBDQTAeFw0xNTEyMTgwOTQyNThaFw0yNTEyMTUwOTQyNThaMBcxFTATBgNV
BAMMDHRlc3RpbmctY2VydDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AMFgCiSGP5DtIxEKM7E9mb4ISUJjN1TOsCdschWUlpXnH06xZBGex4A+ONsxnxp1
06fZlXOfifnoNGh/X3tAV7pw7zGmMnY0yzHdLagnRWv027EQ7SZrhoWqz2Wyo/9b
LCC/XAFLhfqhv4p4EJ6Jszl2j3KaNu87dfFLcmOLGTDM1wJRue6eIwldDpSR0g/Q
nim05CLH9egE8LRNLvsSZD43qxLKp8L9RTJBTSY1VWrbOL8ImqKsSDTE2CpylP8x
B8Ag3ykadMymMUVa2I+WFLu9GuUYFz9BV5V4HyzIRCXNUGSHMKP6v7/PJ02FwE8S
j/3B7DNzvCVCuwFAzGDWQO8CAwEAAaOBlzCBlDAJBgNVHRMEAjAAMB0GA1UdDgQW
BBS3P+ygRrcsJidpq07mq33nhglrFTBGBgNVHSMEPzA9gBQZQfvLZub+U7xuMCcm
0BKs8mPGIKEapBgwFjEUMBIGA1UEAwwLRWFzeS1SU0EgQ0GCCQCLYbb5jQ/cVDAT
BgNVHSUEDDAKBggrBgEFBQcDATALBgNVHQ8EBAMCBaAwDQYJKoZIhvcNAQELBQAD
ggEBADxUgoziTYhp1ZewW30dmC9YQQ9VwOgD98tveRMZnIu3gDgM/f4x7JV9J1JR
Ma/9e3emQbh9JfWnvbdLXZKZLpakRzywCXuZtWu5qbdno7FSh6pXgfrt6apw8+vZ
Z+ccXWH6gaH9EvJWxi7f1bOLeH0wd/tsJgP5HCoUaXHfnt3Bw4/T0+8SiVe7MMuu
mIlx2pM9Tuobxa1xtKNdyZ6yV+FcOdJTkS/g4aHW8D7/zAzrQtOxwzXPb1KQVW3C
C2BwPNVcEnpRPbnpgoYrCIgjnZAPhfu7tMq2OO4/WTb9eZaqDy03Pe0bHenHy5eZ
xZAcBdSQFxeftSwd2BOXB4w/UY0=
-----END CERTIFICATE-----
`)

var keyPEM = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAwWAKJIY/kO0jEQozsT2ZvghJQmM3VM6wJ2xyFZSWlecfTrFk
EZ7HgD442zGfGnXTp9mVc5+J+eg0aH9fe0BXunDvMaYydjTLMd0tqCdFa/TbsRDt
JmuGharPZbKj/1ssIL9cAUuF+qG/ingQnomzOXaPcpo27zt18UtyY4sZMMzXAlG5
7p4jCV0OlJHSD9CeKbTkIsf16ATwtE0u+xJkPjerEsqnwv1FMkFNJjVVats4vwia
oqxINMTYKnKU/zEHwCDfKRp0zKYxRVrYj5YUu70a5RgXP0FXlXgfLMhEJc1QZIcw
o/q/v88nTYXATxKP/cHsM3O8JUK7AUDMYNZA7wIDAQABAoIBABYY5G/SC3eDMauj
z85kLKpjhgOZFNyTFdwbb1n59c9BbvluGfJNg5yq/5JEtFqwtjQLECH7TCgLmdmL
HJ0X+C5s81hoFoIdfE7BaJM7kZpJi8VLGt52ERQ7NaH4bPckMwG2/EuFltTSIPIw
0C1drOZXHwNIjhh+Yfbl2Td40LMbskcrdwJ/y/DfZaEkssmDDj9ZmkGs5ossMpFK
SGeJRECBWkNs6uiyr/W5WoXXdngqSdCJbGdmzz7tsDyTGu6UPazJPvhLzyohhaam
aXCbpL2bSfSl1fVyXjjdrZ9m5dtoXVKaeIf7i08aCl0Ad6Tak+srrjyO1ycRfoD4
+mqUtpECgYEA7vyI0Jxhqmp1vCvDlWMTAufwq/fr1E4NeeJixo5MaJtgi5uvNksd
jbkeVmf3ptSaKR1LwULkgPcmmhfEb3bByzQdEyuC0FlMItWYUZHtw9QU0AoD5JF9
u48ESYDsgJzloUneY/mVZGMXPvBZLrS4xvN0M6xmy0BcMblhBawH+hkCgYEAzyRA
RiFtyGqOogEU7kjuJ7yyA1guikTKURCUPegnGv15ZlXOapxLB8GPhahUcAJ8vv+7
7Z3ia1SOt7BSnsIhW55ognpEWw+L+MIjz3WJOzla2P4ohDJZwSy1Cj4lWho+3G65
PLQZ+O0Hb8wjsXNxPjISmopdHHJuoqECFUZjhEcCgYEAwgNZvqF12DddJUoSGbC4
ul85TyKR3WUQI6bZsX/MIBAjrLLS5yzL7UYfjt4QeuuVy1LxMQ/xGZGLUQWCf0rV
wPWptOpZ5HLaEF1+rpndgGEoFExNJL3IaP+N5242kaLN+MZTOK5hzYF0WbAddoFY
kIsMBvcq7E5vih6I2WXzg+ECgYEAoSgnCWkArKiah9gnnKwI/cmFBa0ZqGGUtjUb
4H45znnedYvUqIUoqsQhEW/BIdQNkdwNLfVkLvT+hFMeNH38zfcUgE7315Dk6YjB
q6paNkWNNL2ocBFsWyqZP3rSPKOmvIE4hM3qVwyyeHxuWKTkOetjJfD4OCWfhc+W
e07kJgkCgYB//bWJ9oJYkXcWCaqnAN8Pc/L3UroJZMyd6RFM6T/gpnud9Kqxi7h1
bwxTydMFC7209lr6UfEj0/xmmyf190fvvMFp/R30S/EkXZ89X1hyb9yuWXxkMhgo
DhcasEfsMesLuXWJ2WdABvNy4HdmV+327gF9Xi4atRv+sB3z+laChA==
-----END RSA PRIVATE KEY-----
`)
