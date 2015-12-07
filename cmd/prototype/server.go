package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
	"github.com/skriptble/nine/stream/transport"
)

type xstate interface {
	Next() xstate
}

type xstateConnected struct {
	conn net.Conn
	strm stream.Streamv0
}

type xstateTLSNegotiation struct {
	strm stream.Streamv0
	conn net.Conn
}
type xstateSASLNegotiation struct {
	strm stream.Streamv0
}
type xstateBind struct {
	strm stream.Streamv0
}
type xstateFailure struct{}
type xstateClose struct{}

func (xsc xstateConnected) Next() xstate {
	xsc.strm = stream.Newv0(transport.NewTCPv0(xsc.conn), stream.Receiving)
	// elem, err := xsc.strm.Next()
	// log.Println("elem", elem)
	// log.Printf("%s\n", elem.WriteBytes())
	// if err != nil {
	// 	return xstateClose{}
	// }
	// header, err := stream.NewHeader(elem)
	// log.Println("header", header)
	// log.Printf("%s\n", header.WriteBytes())
	// if err != nil {
	// 	return xstateClose{}
	// }
	// header.To, header.From = header.From, header.To
	// header.ID = genStreamID()
	// b := header.WriteBytes()
	// xsc.strm.Write(b)

	xsc.strm.Start()
	// TODO(skriptble): Should this be a proper Element?
	elem := element.Element{Space: "stream", Tag: "features"}
	stls := element.StartTLS
	stls.Child = append(stls.Child, element.Element{Tag: "required"})
	elem.Child = append(elem.Child, stls)
	b := []byte(`<stream:features><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'><required/></starttls></stream:features>`)
	xsc.strm.Write(b)
	eb := elem.WriteBytes()
	log.Printf("\n1:%s\n2:%s\n", b, eb)
	log.Println(reflect.DeepEqual(b, eb))
	// xsc.strm.Write(eb)
	return xstateTLSNegotiation{strm: xsc.strm, conn: xsc.conn}
}

func (xsc xstateTLSNegotiation) Next() xstate {
	elem, err := xsc.strm.Next()
	if err != nil {
		return xstateClose{}
	}
	log.Printf("%s\n", elem.WriteBytes())
	if elem.Tag != "starttls" {
		return xstateClose{}
	}
	ns := elem.SelectAttrValue("xmlns", "")
	if ns != "urn:ietf:params:xml:ns:xmpp-tls" {
		return xstateClose{}
	}
	xsc.strm.Write([]byte("<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>"))
	tlsConn := tls.Server(xsc.conn, TLSConfig)
	tlsConn.Handshake()

	xsc.conn = net.Conn(tlsConn)
	xsc.strm = stream.Newv0(transport.NewTCPv0(xsc.conn), stream.Receiving)
	return xstateSASLNegotiation{strm: xsc.strm}
}

func (xsc xstateSASLNegotiation) Next() xstate {
	elem, err := xsc.strm.Next()
	log.Println("elem", elem)
	if err != nil {
		return xstateClose{}
	}
	header, err := stream.NewHeader(elem)
	log.Println("header", header)
	if err != nil {
		return xstateClose{}
	}
	header.To, header.From = header.From, header.To
	header.ID = genStreamID()
	b := header.WriteBytes()
	xsc.strm.Write(b)

	b = []byte("<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>PLAIN</mechanism></mechanisms></stream:features>")
	xsc.strm.Write(b)

	elem, err = xsc.strm.Next()
	log.Println("elem", elem)
	if err != nil {
		return xstateClose{}
	}
	log.Printf("%s\n", elem.WriteBytes())
	data, err := base64.StdEncoding.DecodeString(elem.Text())
	if err != nil {
		return xstateClose{}
	}
	fmt.Printf("%q\n", data)
	b = []byte("<success xmlns='urn:ietf:params:xml:ns:xmpp-sasl'/>")
	xsc.strm.Write(b)
	return xstateBind{strm: xsc.strm}
}

func (xsb xstateBind) Next() xstate {
	elem, err := xsb.strm.Next()
	log.Printf("%+v\n%s", elem, elem.WriteBytes())
	if err != nil {
		return xstateClose{}
	}
	header, err := stream.NewHeader(elem)
	log.Printf("%+v\n%s", header, header.WriteBytes())
	if err != nil {
		return xstateClose{}
	}
	header.To, header.From = header.From, header.To
	header.To = "skriptble@localhost"
	header.ID = genStreamID()
	b := header.WriteBytes()
	xsb.strm.Write(b)

	b = []byte("<stream:features><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></stream:features>")
	xsb.strm.Write(b)
	elem, err = xsb.strm.Next()
	log.Printf("%+v\n%s", elem, elem.WriteBytes())
	if err != nil {
		return xstateClose{}
	}

	return nil
}

func (xsc xstateClose) Next() xstate {
	return nil
}

func runFSM(xs xstate) {
	for xs != nil {
		xs = xs.Next()
	}
}

func genStreamID() string {
	id := make([]byte, 16)
	rand.Read(id)

	id[8] = (id[8] | 0x80) & 0xBF
	id[6] = (id[6] | 0x40) & 0x4F

	return fmt.Sprintf("%x", id)
}
