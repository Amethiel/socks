package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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
		return errors.New(fmt.Sprintf("Message parse error: %v", buff))
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
		fmt.Println(err)
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
		} else {
			conn.Write([]byte{5, 1})
			return nil, fmt.Errorf("CMD(%X) not supported", buff[1])
		}
	} else {
		conn.Write([]byte{5, 1})
		return nil, fmt.Errorf("Message parse error: %v", buff)
	}

	return nil, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	err := methodSelection(conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	proxyConn, err := connect(conn)
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
	cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", ":1443", config)
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
