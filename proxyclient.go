package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	cert, err := tls.LoadX509KeyPair(clientPem, clientKey)
	if err != nil {
		log.Fatalln("FAILED to load client key:(%+v, %+v), error message: %s", clientPem, clientKey, err)
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS12,
		ClientCAs:          pool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: true,
	}
	proxyConn, err := tls.Dial("tcp", remote, config)
	if err != nil {
		log.Fatalf("FAILED to connect to remote %+v, error message: %s", remote, err)
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
	exePath   string
	pool      *x509.CertPool
	remote    string
	caPem     string
	clientPem string
	clientKey string
	port      int
)

func init() {
	flag.StringVar(&remote, "r", "proxy.focusworks.net:1443", "远程服务地址")
	flag.StringVar(&caPem, "ca", "ca.pem", "CA证书")
	flag.StringVar(&clientPem, "c", "client.pem", "客户端证书")
	flag.StringVar(&clientKey, "k", "client.key", "客户端私钥")
	flag.IntVar(&port, "p", 2080, "本地监听端口号")
	flag.Parse()

	dir, _ := os.Executable()
	exePath = filepath.Dir(dir)

	pool = x509.NewCertPool()
	caCrt, err := ioutil.ReadFile(caPem)
	if err != nil {
		log.Fatalln("ReadFile err:", err)
	}
	pool.AppendCertsFromPEM(caCrt)
}

func main() {
	log.Println("Proxy start ... ")
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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
