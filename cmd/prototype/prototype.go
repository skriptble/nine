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
	"github.com/skriptble/nine/stream/features"
	"github.com/skriptble/nine/stream/transport"
)

// type State interface {
// 	Next() State
// }

type state func(c net.Conn) state

var TLSConfig *tls.Config
var mechs = map[string]features.SASLMechanism{
	"PLAIN": features.SASLPlain{},
}

var fn features.FeatureNegotiator

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
		RootCAs:      certpool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   "localhost",
	}

	fn = features.NewFeatureNegotiator(Holder{}, []features.Step{
		features.Step{"starttls": features.NewStartTLS(TLSConfig, true)},
		features.Step{"auth": features.NewSASL(true, 3, mechs)},
		features.Step{"iq": features.NewBind()},
	})

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
		// start := xstateConnected{conn: conn}
		// go runFSM(start)

		// log.Println("Starting new connection state machine")
		// s := stream.Newv0(transport.NewTCPv0(conn), stream.Receiving)
		// go run(fn, s)

		saslHandler := sasl.NewHandler(map[string]sasl.Mechanism{
			"PLAIN": sasl.NewPlainMechanism(sasl.FakePlain{}),
		})
		bindHandler := bind.NewHandler()
		sessionHandler := bind.NewSessionHandler()
		fhs := []stream.FeatureHandler{
			saslHandler,
			bindHandler,
			// sessionHandler,
		}
		ehs := []stream.ElementHandler{
			{Tag: "auth", Space: namespace.SASL,
				FSM: saslHandler},
		}
		ihs := []stream.IQHandler{
			{
				Tag: "bind", Space: namespace.Bind,
				Type: string(stanza.IQSet),
				FSM:  bindHandler,
			},
			{
				Tag: "session", Space: namespace.Session,
				Type: string(stanza.IQSet),
				FSM:  sessionHandler,
			},
			// This should always be last.
			{
				Tag: "*", Space: "*",
				Type: "*",
				FSM:  stream.NewHandler(),
			},
		}
		mhs := []stream.MessageHandler{}
		phs := []stream.PresenceHandler{}

		tp := transport.NewTCP(conn, stream.Receiving, TLSConfig, true)
		props := stream.NewProperties()
		s := stream.New(tp, props, stream.Receiving, true).
			AddFeatureHandlers(fhs...).
			AddElementHandlers(ehs...).
			AddIQHandlers(ihs...).
			AddMessageHandlers(mhs...).
			AddPresenceHandlers(phs...)
		go s.Run()
	}
}

func run(state stream.FSMv0, s stream.Streamv0) {
	for state != nil {
		state, s = state.Next(s)
	}
}

type Holder struct{}

func (h Holder) Next(s stream.Streamv0) (stream.FSMv0, stream.Streamv0) {
	el, _ := s.Next()
	iq, _ := stanza.TransformIQ(el)

	res := stanza.IQ{
		stanza.Stanza{
			ID:   iq.ID,
			Type: "result",
			To:   "skriptble@localhost/FullStack-WebDev-Pro",
			From: "localhost",
		},
	}
	s.WriteElement(res.TransformElement())

	el, _ = s.Next()
	iq, _ = stanza.TransformIQ(el)

	res = stanza.IQ{
		stanza.Stanza{
			ID:   iq.ID,
			Type: "result",
			To:   "skriptble@localhost/FullStack-WebDev-Pro",
			From: "localhost",
		},
	}
	s.WriteElement(res.TransformElement())

	el, _ = s.Next()
	iq, _ = stanza.TransformIQ(el)

	res = stanza.IQ{
		stanza.Stanza{
			ID:   iq.ID,
			Type: "result",
			To:   "skriptble@localhost/FullStack-WebDev-Pro",
			From: "localhost",
		},
	}
	s.WriteElement(res.TransformElement())

	for {
		s.Next()
	}
	return nil, s
}

// func run(c net.Conn) {
// 	var s state
// 	for s = begin; s != nil; {
// 		s = s(c)
// 	}
// }
//
// func begin(conn net.Conn) state {
// 	r := bufio.NewReader(conn)
// 	str, err := r.ReadString('\n')
// 	if err != nil {
// 		if err.Error() == "EOF" {
// 			defer conn.Close()
// 			return nil
// 		}
// 		log.Println(err)
// 	}
// 	switch {
// 	case strings.HasPrefix(str, "state two"):
// 		return two
// 	case strings.TrimSpace(str) == "exit":
// 		defer conn.Close()
// 		return nil
// 	}
// 	fmt.Println(str)
// 	return begin
// }
//
// func two(conn net.Conn) state {
// 	_, err := conn.Write([]byte("Entered state two!\n"))
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	return begin
// }
