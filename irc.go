package main

import (
	"crypto/tls"
	"fmt"

	"github.com/fluffle/goirc/client"
)

func newConn(host string, port int, ssl bool, nick string, join string) *client.Conn {
	cfg := client.NewConfig(nick)
	if ssl {
		cfg.SSL = true
		cfg.SSLConfig = &tls.Config{ServerName: host, InsecureSkipVerify: true}
		cfg.NewNick = func(n string) string { return n + "^" }
	}
	cfg.Server = fmt.Sprintf("%s:%d", host, port)
	irc := client.Client(cfg)

	irc.HandleFunc(client.CONNECTED, func(c *client.Conn, l *client.Line) {
		irc.Join(join)
	})

	return irc
}
