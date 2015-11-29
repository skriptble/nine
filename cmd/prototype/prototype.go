package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// type State interface {
// 	Next() State
// }

type state func(c net.Conn) state

func main() {
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
		go run(conn)
	}
}

func run(c net.Conn) {
	var s state
	for s = begin; s != nil; {
		s = s(c)
	}
}

func begin(conn net.Conn) state {
	r := bufio.NewReader(conn)
	str, err := r.ReadString('\n')
	if err != nil {
		if err.Error() == "EOF" {
			defer conn.Close()
			return nil
		}
		log.Println(err)
	}
	switch {
	case strings.HasPrefix(str, "state two"):
		return two
	case strings.TrimSpace(str) == "exit":
		defer conn.Close()
		return nil
	}
	fmt.Println(str)
	return begin
}

func two(conn net.Conn) state {
	_, err := conn.Write([]byte("Entered state two!\n"))
	if err != nil {
		log.Println(err)
	}
	return begin
}
