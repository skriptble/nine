package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"

	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/stream"
	"github.com/skriptble/nine/stream/transport"
)

var config *tls.Config

func init() {
	// turn on debugging
	stream.Trace.SetOutput(os.Stderr)
	stream.Debug.SetOutput(os.Stderr)
}

func init() {
	var certpool = x509.NewCertPool()
	caCert, err := ioutil.ReadFile("./ca.crt")
	if err != nil {
		panic(err)
	}
	if !certpool.AppendCertsFromPEM(caCert) {
		panic("Could not append CA Certificate")
	}

	config = &tls.Config{
		RootCAs:            certpool,
		InsecureSkipVerify: true,
		ServerName:         "localhost",
	}
}
func main() {
	// Establish TCP connection
	// |--Create stream handler
	// |-- In the TCP transport, handle stream features and upgrade the
	// |   underlying TCP connection when starttls is found.
	// Authenticate via SASL
	// Bind resource
	conn, err := net.Dial("tcp", "127.0.0.1:5222")
	if err != nil {
		panic(err)
	}

	tsp := transport.NewReceivingTCP(conn, stream.Initiating, config, true)
	fm := stream.NewFeaturesMux()
	strm := stream.New(tsp, fm, stream.Initiating)
	strm.Header = stream.Header{
		From:      "client@localhost",
		To:        "localhost",
		Namespace: namespace.Client,
		Version:   "1.0",
	}
	strm.Run()
}
