package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	proxyConn, err := net.Dial("tcp", "192.168.100.140:1443")
	if err != nil {
		fmt.Println(err)
		return
	}

	ExitChan := make(chan bool, 1)
	go func(sconn net.Conn, dconn net.Conn) {
		io.Copy(sconn, dconn)
		ExitChan <- true
	}(conn, proxyConn)
	go func(sconn net.Conn, dconn net.Conn) {
		io.Copy(sconn, dconn)
		ExitChan <- true
	}(proxyConn, conn)

	<-ExitChan
}

func main() {
	fmt.Print("server start ... ")
	ln, err := net.Listen("tcp", ":2080")
	if err != nil {
		fmt.Println("FAILED")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("OK")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}
