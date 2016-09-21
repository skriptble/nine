package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/skriptble/nine/bind"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/sasl"
	"github.com/skriptble/nine/stream"
	"github.com/skriptble/nine/stream/transport"
)

type state func(c net.Conn) state

var TLSConfig *tls.Config

func init() {
	// turn on debugging
	stream.Trace.SetOutput(os.Stderr)
	stream.Debug.SetOutput(os.Stderr)
}

func main() {
	var certpool = x509.NewCertPool()
	caCert, err := ioutil.ReadFile("./tlsupgrade/ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	if !certpool.AppendCertsFromPEM(caCert) {
		log.Fatal("Could not append CA Certificate")
	}

	cert, err := tls.LoadX509KeyPair("./tlsupgrade/localhost.crt", "./tlsupgrade/localhost.unencrypted.pem")
	if err != nil {
		log.Fatal(err)
	}

	TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		// RootCAs:      certpool,
		ClientAuth: tls.VerifyClientCertIfGiven,
		ServerName: "localhost",
	}

	ln, err := net.Listen("tcp", ":5222")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}

		saslHandler := sasl.NewHandler(map[string]sasl.Mechanism{
			"PLAIN": sasl.NewPlainMechanism(sasl.FakePlain{}, "localhost"),
		})
		bindHandler := bind.NewHandler()
		sessionHandler := bind.NewSessionHandler()
		iqHandler := stream.NewIQMux().
			Handle(namespace.Bind, "bind", string(stanza.IQSet), bindHandler).
			Handle(namespace.Session, "session", string(stanza.IQSet), sessionHandler)

		if iqHandler.Err() != nil {
			log.Fatal(iqHandler.Err())
		}

		elHandler := stream.ElementMuxV2{}.
			Handle(namespace.SASL, "auth", saslHandler).
			Handle(namespace.SASL, "response", saslHandler).
			Handle(namespace.Client, "iq", iqHandler).
			Handle(namespace.Client, "presence", stream.Blackhole{}).
			Handle(namespace.Client, "message", stream.Blackhole{})

		if elHandler.Err() != nil {
			log.Fatal(iqHandler.Err())
		}

		fgs := []stream.FeatureGenerator{
			saslHandler,
			bindHandler,
			// sessionHandler,
		}

		tp := transport.NewReceivingTCP(conn, TLSConfig, true, "localhost", fgs)
		props := stream.NewProperties()
		props.Domain = "localhost"
		s := stream.New(tp, elHandler, stream.Receiving)
		bindHandler.RegisterStream(s)
		go s.Run()
	}
}
