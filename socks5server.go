package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
)

func methodSelection(conn net.Conn) error {
	// version identifier/method selection
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+

	methodSelectionLength := 1 + 1 + 255
	buff := make([]byte, methodSelectionLength)
	_, err := conn.Read(buff)
	if err != nil {
		conn.Write([]byte{5, 0xFF})
		return err
	}

	if buff[0] == 5 {
		conn.Write([]byte{5, 0})
	} else {
		conn.Write([]byte{5, 0xFF})
		return fmt.Errorf("Message parse error: %v", buff)
	}

	return nil
}

func connect(conn net.Conn) (net.Conn, error) {
	// cmd: connect
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	//
	// DST.ADDR(IP V4 address)
	// +--------------+
	// |IP V4 address |
	// +--------------+
	// |      4       |
	// +--------------+
	//
	// DST.ADDR(DOMAINNAME)
	// +-------------------+-----------------------------+
	// |DOMAIN NAME LENGTH | FULLY-QUALIFIED DOMAIN NAME |
	// +-------------------+-----------------------------+
	// |         1         |         1 to 255            |
	// +-------------------+-----------------------------+
	//
	// DST.ADDR(IP V6 address)
	// +--------------+
	// |IP V6 address |
	// +--------------+
	// |      16      |
	// +--------------+
	//
	//
	// resp
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	connectLength := 1 + 1 + 1 + 1 + 1 + 255 + 2
	buff := make([]byte, connectLength)
	_, err := conn.Read(buff)
	if err != nil {
		//log.Println("FAILED to read data from client", err)
		conn.Write([]byte{5, 1})
		return nil, nil
	}

	if buff[0] == 5 {
		if buff[1] == 1 {
			// only support ipv4 address
			targetAddr := net.IP(buff[4:8])
			targetPort := int(buff[8])<<8 + int(buff[9])
			addrStr := fmt.Sprintf("%s:%d", targetAddr, targetPort)
			proxyConn, err := net.Dial("tcp", addrStr)
			if err != nil {
				conn.Write([]byte{5, 5})
				return nil, err
			}

			outBuff := []byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}
			connAddr := conn.LocalAddr().(*net.TCPAddr)
			copy(outBuff[4:8], connAddr.IP.To4())
			binary.Write(bytes.NewBuffer(outBuff[8:10]), binary.BigEndian, connAddr.Port)
			conn.Write(outBuff)
			return proxyConn, nil
		}

		conn.Write([]byte{5, 1})
		return nil, fmt.Errorf("CMD(%X) not supported", buff[1])
	}

	conn.Write([]byte{5, 1})
	return nil, fmt.Errorf("Message parse error: %v", buff)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	err := methodSelection(conn)
	if err != nil {
		log.Println("FAILED to select authentication method", err)
		return
	}

	proxyConn, err := connect(conn)
	if err != nil {
		log.Println("FAILED to connect to target server", err)
		return
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
	caPem     string
	serverPem string
	serverKey string
	port      int
)

func init() {
	flag.StringVar(&caPem, "ca", "ca.pem", "CA证书")
	flag.StringVar(&serverPem, "c", "server.pem", "服务端证书")
	flag.StringVar(&serverKey, "k", "server.key", "服务端私钥")
	flag.IntVar(&port, "p", 1443, "监听端口号")
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
	log.Println("Server start ... ")
	//log.Println(exePath)

	cert, err := tls.LoadX509KeyPair(serverPem, serverKey)
	if err != nil {
		log.Fatalln("FAILED to load server key", err)
		return
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS12,
		ClientCAs:          pool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: true,
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), config)
	if err != nil {
		log.Fatalln("FAILED to start server", err)
	}
	log.Println("Server start OK")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}
