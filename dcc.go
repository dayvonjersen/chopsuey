package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func ntohl(n int) net.IP {
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

func dccHandler(servConn *serverConnection, who, req string) {
	args := strings.Split(req, " ")
	switch args[0] {
	case "SEND":
		filename := args[1]
		ipnl, _ := strconv.Atoi(args[2])
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
