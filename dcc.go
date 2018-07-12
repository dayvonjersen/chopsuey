package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func ntohl(n int64) net.IP {
	d := (n & 0xff) << 24
	c := (n & 0xff00) << 8
	b := (n & 0xff0000) >> 8
	a := (n & 0xff000000) >> 24
	return net.IPv4(
		byte(a&0xff),
		byte((b>>8)&0xff),
		byte((c>>16)&0xff),
		byte((d>>24)&0xff),
	)
}

func htonl(ip net.IP) int64 {
	a, b, c, d := int64(ip[0]), int64(ip[1]), int64(ip[2]), int64(ip[3])
	return a<<24 | b<<16 | c<<8 | d
}

func wanIP() (net.IP, error) {
	resp, err := http.Get("http://icanhazip.com")

	if err != nil {
		return nil, err
	}

	buf := make([]byte, 16)
	resp.Body.Read(buf)

	i := 15
	for ; buf[i] != '\n'; i-- {
	}

	return net.ParseIP(string(buf[:i])), nil
}

func localIP() (net.IP, error) {
	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, ifaceAddr := range ifaceAddrs {
		ifaceIP := net.ParseIP(strings.Split(ifaceAddr.String(), "/")[0])
		if ifaceIP != nil {
			if ifaceIP.To4() != nil {
				return ifaceIP.To4(), nil
			}
		}
	}
	return nil, fmt.Errorf("couldn't bind to an ip4 address on the local machine")
}

func fileTransfer(servConn *serverConnection, who, filename string) {
	f, err := os.Open(filename)
	checkErr(err)
	stat, err := f.Stat()
	checkErr(err)
	filesize := stat.Size()

	if servConn.ip == nil {
		servConn.ip, _ = localIP()
	}
	ip := servConn.ip

	ln, err := net.Listen("tcp", ip.String()+":0")
	checkErr(err)
	go func() {
		for {
			conn, err := ln.Accept()
			checkErr(err)
			io.Copy(conn, f)
			f.Close()
			conn.Close()
			return
		}
	}()
	addr := strings.Split(ln.Addr().String(), ":")
	log.Println("listening on", addr)

	ipnl := htonl(ip)

	port, _ := strconv.ParseInt(addr[1], 10, 64)

	ctcpMsg := fmt.Sprintf("DCC SEND %s %v %v %v", filename, ipnl, port, filesize)
	log.Println(ctcpMsg)

	servConn.conn.Ctcp(who, ctcpMsg)
}

func dccHandler(servConn *serverConnection, who, req string) {
	args := strings.Split(req, " ")
	switch args[0] {
	case "SEND":
		filename := args[1]
		ipnl, _ := strconv.ParseInt(args[2], 10, 64)
		ip := ntohl(ipnl)
		port, _ := strconv.Atoi(args[3])
		filesize, _ := strconv.Atoi(args[4])

		log.Println("got DCC SEND from", who, ":", filename, ip, port, filesize)

		servConn.conn.Ctcp(who, "DCC ACCEPT", filename, fmt.Sprintf("%d", port), "0")
		f, err := os.Create(filename)
		checkErr(err)
		defer f.Close()

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
		checkErr(err)

		size := 0
		for {
			b := make([]byte, 32)
			n, err := conn.Read(b)
			size += n
			if err == io.EOF || size >= filesize {
				conn.Close()
				break
			} else if err != nil {
				log.Println(err)
				break
			}
		}
		log.Println(filesize, size, err)
		log.Println(filename, ": file transfer complete")
	default:
		log.Println("got DCC request but did not process:", req)
	}
}
