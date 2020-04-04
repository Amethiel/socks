package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func handleConnection(remote string, conn net.Conn) {
	defer conn.Close()

	config := &tls.Config{InsecureSkipVerify: true}
	proxyConn, err := tls.Dial("tcp", remote, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	ExitChan := make(chan bool)
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
	var remote string
	flag.StringVar(&remote, "r", "127.0.0.1:1443", "远程服务地址，默认值：127.0.0.1:1443")

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
		go handleConnection(remote, conn)
	}
}
