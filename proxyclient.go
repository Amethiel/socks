package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	cert, err := tls.LoadX509KeyPair("client1.pem", "client1.key")
	if err != nil {
		log.Fatalln("FAILED to load client key", err)
	}

	//config := &tls.Config{InsecureSkipVerify: true}
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS12,
		ClientCAs:          pool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: true,
	}
	proxyConn, err := tls.Dial("tcp", remote, config)
	if err != nil {
		log.Fatalln("FAILED to connect to remote", err)
	}

	ExitChan := make(chan bool)
	go func(sconn net.Conn, dconn net.Conn) {
		if sconn != nil && dconn != nil {
			io.Copy(sconn, dconn)
		}
		ExitChan <- true
	}(conn, proxyConn)
	go func(sconn net.Conn, dconn net.Conn) {
		if sconn != nil && dconn != nil {
			io.Copy(sconn, dconn)
		}
		ExitChan <- true
	}(proxyConn, conn)

	<-ExitChan
}

var (
	exePath string
	pool    *x509.CertPool
	remote  string
)

func init() {
	flag.StringVar(&remote, "r", "proxy.focusworks.net:1443", "远程服务地址")

	dir, _ := os.Executable()
	exePath = filepath.Dir(dir)

	pool = x509.NewCertPool()
	caCrt, err := ioutil.ReadFile("ca.pem")
	if err != nil {
		log.Fatalln("ReadFile err:", err)
	}
	pool.AppendCertsFromPEM(caCrt)
}

func main() {
	flag.Parse()

	log.Println("Proxy start ... ")
	ln, err := net.Listen("tcp", ":2080")
	if err != nil {
		log.Fatalln("FAILED to start proxy", err)
	}
	log.Println("Proxy start OK")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}
