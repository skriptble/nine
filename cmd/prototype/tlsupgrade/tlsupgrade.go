package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"strings"
)

var TLSConfig *tls.Config

func main() {
	var certpool = x509.NewCertPool()
	caCert, err := ioutil.ReadFile("./ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	if !certpool.AppendCertsFromPEM(caCert) {
		log.Fatal("Could not append CA Certificate")
	}

	cert, err := tls.LoadX509KeyPair("./localhost.crt", "./localhost.unencrypted.pem")
	if err != nil {
		log.Fatal(err)
	}

	TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certpool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   "localhost",
	}

	ln, err := net.Listen("tcp", ":2555")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}
		go run(conn)
	}
}

func run(c net.Conn) {
	var tlsConn *tls.Conn
	var upgrade bool
	reader := bufio.NewReader(c)
	tp := textproto.NewReader(reader)

	defer c.Close()

	// for {
	// 	log.Println("Reading byte")
	// 	byt, err := reader.ReadString('\n')
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			log.Println("Oops")
	// 			return
	// 		}
	// 		log.Println(err)
	// 	}
	// 	log.Printf("%s", byt)
	// }
	c.Write([]byte("220 mail.example.org ESMTP service ready\n"))
	for {
		line, err := tp.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Println("error", err)
		}
		log.Println(line)
		if upgrade {
			upgrade = false
		}
		if line == "STARTTLS" {
			log.Println("Starting TLS!")
			c.Write([]byte("220 Go Ahead\n"))
			upgrade = true
			tlsConn = tls.Server(c, TLSConfig)
			tlsConn.Handshake()

			c = net.Conn(tlsConn)
		}
		if strings.HasPrefix(line, "EHLO") {
			c.Write([]byte("250-mail.example.org offers a warm hug of welcome\n"))
			c.Write([]byte("250 STARTTLS\n"))
		}
	}
}
