package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	goirc "github.com/fluffle/goirc/client"
)

type serverConnection struct {
	cfg       *clientConfig
	conn      *goirc.Conn
	chatBoxes map[string]*chatBox
	newChats  chan string
}

func (servConn *serverConnection) connect() {
	checkErr(servConn.conn.ConnectTo(servConn.cfg.Host))
}

type chatBox struct {
	printMessage func(nick, msg string)
	sendMessage  func(msg string)
	setNickList  func(nicks []string)
}

func newServerConnection(cfg *clientConfig) *serverConnection {
	goircCfg := goirc.NewConfig(cfg.Nick)
	if cfg.Ssl {
		goircCfg.SSL = true
		goircCfg.SSLConfig = &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: true,
		}
		goircCfg.NewNick = func(n string) string { return n + "^" }
	}
	goircCfg.Server = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn := goirc.Client(goircCfg)

	servConn := &serverConnection{
		cfg:       cfg,
		conn:      conn,
		chatBoxes: map[string]*chatBox{},
		newChats:  make(chan string),
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		for _, channel := range cfg.Autojoin {
			conn.Join(channel)
			servConn.chatBoxes[channel] = &chatBox{}
			servConn.newChats <- channel
		}
	})

	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		chat, ok := servConn.chatBoxes[l.Args[0]]
		if !ok {
			log.Printf("%#v", l)
			channel := l.Args[0]
			if channel == servConn.cfg.Nick {
				channel = l.Nick
			}
			servConn.chatBoxes[channel] = &chatBox{}
			servConn.newChats <- channel
			chat = servConn.chatBoxes[channel]
		}
		chat.printMessage(l.Nick, l.Args[1])
	})

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		chat, ok := servConn.chatBoxes[l.Args[2]]
		if !ok {
			log.Println("got 353 but user not on channel:", l.Args[2])
			return
		}
		nicks := []string{}
		for _, nick := range strings.Split(l.Args[3], " ") {
			if nick != "" {
				nicks = append(nicks, nick)
			}
		}
		chat.setNickList(nicks)
	})

	return servConn
}

// temporary until we handle all unhandled server response codes

type tsoLogger struct {
	LogFn func(string)
}

func (l *tsoLogger) Debug(f string, a ...interface{}) {
	l.LogFn(fmt.Sprintf(f, a...))
}
func (l *tsoLogger) Info(f string, a ...interface{})  {}
func (l *tsoLogger) Warn(f string, a ...interface{})  {}
func (l *tsoLogger) Error(f string, a ...interface{}) {}
