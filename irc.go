package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	goirc "github.com/fluffle/goirc/client"
	"github.com/lxn/walk"
)

type serverConnection struct {
	cfg        *clientConfig
	conn       *goirc.Conn
	chatBoxes  map[string]*chatBox
	newChats   chan string
	closeChats chan *chatBox
}

func (servConn *serverConnection) connect() {
	checkErr(servConn.conn.ConnectTo(servConn.cfg.Host))
}

func (servConn *serverConnection) join(channel string) {
	servConn.conn.Join(channel)
	servConn.chatBoxes[channel] = newChatBox()
	servConn.newChats <- channel
}

func (servConn *serverConnection) part(channel, reason string) {
	if channel[0] == '#' {
		servConn.conn.Part(channel, reason)
	}
	chat, ok := servConn.chatBoxes[channel]
	if !ok {
		log.Panicln("user not on channel:", channel)
	}
	close(chat.messages)
	servConn.closeChats <- chat
	delete(servConn.chatBoxes, channel)
}

type chatBox struct {
	printMessage   func(msg string)
	sendMessage    func(msg string)
	updateNickList func()
	nickList       *nickList
	nickListUpdate chan struct{}
	messages       chan string
	tabPage        *walk.TabPage
}

func newChatBox() *chatBox {
	return &chatBox{messages: make(chan string), nickList: &nickList{}, nickListUpdate: make(chan struct{})}
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
		cfg:        cfg,
		conn:       conn,
		chatBoxes:  map[string]*chatBox{},
		newChats:   make(chan string),
		closeChats: make(chan *chatBox),
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		for _, channel := range cfg.Autojoin {
			servConn.join(channel)
		}
	})

	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		if channel == servConn.cfg.Nick {
			channel = l.Nick
		}
		chat, ok := servConn.chatBoxes[channel]
		if !ok {
			servConn.chatBoxes[channel] = newChatBox()
			servConn.newChats <- channel
			chat = servConn.chatBoxes[channel]
		}
		chat.messages <- fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), l.Nick, l.Args[1])
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		if channel == servConn.cfg.Nick {
			channel = l.Nick
		}
		chat, ok := servConn.chatBoxes[channel]
		if !ok {
			servConn.chatBoxes[channel] = newChatBox()
			servConn.newChats <- channel
			chat = servConn.chatBoxes[channel]
		}
		chat.messages <- fmt.Sprintf("%s * %s %s", time.Now().Format("15:04"), l.Nick, l.Args[1])
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		if channel == servConn.cfg.Nick {
			channel = l.Nick
		}
		if channel == "AUTH" && servConn.cfg.Nick != "AUTH" {
			// servers commonly send these NOTICEs when connecting:
			//
			// :irc.example.org NOTICE AUTH :*** Looking up your hostname...
			// :irc.example.org NOTICE AUTH :*** Found your hostname
			//
			// dropping these messages for now...
			return
		}
		chat, ok := servConn.chatBoxes[channel]
		if !ok {
			servConn.chatBoxes[channel] = newChatBox()
			servConn.newChats <- channel
			chat = servConn.chatBoxes[channel]
		}
		chat.messages <- fmt.Sprintf("%s *** %s: %s", time.Now().Format("15:04"), l.Nick, l.Args[1])
	})

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		chat, ok := servConn.chatBoxes[l.Args[2]]
		if !ok {
			log.Println("got 353 but user not on channel:", l.Args[2])
			return
		}

		for _, nick := range strings.Split(l.Args[3], " ") {
			if nick != "" {
				chat.nickList.Add(nick)
			}
		}
		chat.nickListUpdate <- struct{}{}
	})

	conn.HandleFunc(goirc.JOIN, func(c *goirc.Conn, l *goirc.Line) {
		chat, ok := servConn.chatBoxes[l.Args[0]]
		if !ok {
			log.Println("got JOIN but user not on channel:", l.Args[0])
			return
		}
		chat.nickList.Add(l.Nick)
		chat.nickListUpdate <- struct{}{}
		chat.messages <- "* " + l.Nick + " has joined " + l.Args[0]
	})

	conn.HandleFunc(goirc.PART, func(c *goirc.Conn, l *goirc.Line) {
		chat, ok := servConn.chatBoxes[l.Args[0]]
		if !ok {
			log.Println("got PART but user not on channel:", l.Args[0])
			return
		}
		chat.nickList.Remove(l.Nick)
		chat.nickListUpdate <- struct{}{}
		chat.messages <- "** " + l.Nick + " has left " + l.Args[0]
	})

	conn.HandleFunc(goirc.QUIT, func(c *goirc.Conn, l *goirc.Line) {
		for _, chat := range servConn.chatBoxes {
			if chat.nickList.Has(l.Nick) {
				chat.messages <- "** " + l.Nick + " has quit: " + l.Args[0]
				chat.nickList.Remove(l.Nick)
				chat.nickListUpdate <- struct{}{}
			}
		}
	})

	conn.HandleFunc(goirc.NICK, func(c *goirc.Conn, l *goirc.Line) {
		if l.Nick == servConn.cfg.Nick {
			servConn.cfg.Nick = l.Args[0]
		}
		for _, chat := range servConn.chatBoxes {
			if chat.nickList.Has(l.Nick) {
				chat.messages <- "** " + l.Nick + " is now known as " + l.Args[0]
				chat.nickList.Remove(l.Nick)
				chat.nickList.Add(l.Args[0])
				chat.nickListUpdate <- struct{}{}
			}
		}
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
