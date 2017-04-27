package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	goirc "github.com/fluffle/goirc/client"
)

type serverConnection struct {
	cfg          *clientConfig
	conn         *goirc.Conn
	chatBoxes    []*chatBox
	newChats     chan string
	newChatBoxes chan *chatBox
	closeChats   chan *chatBox
}

func (servConn *serverConnection) connect() {
	checkErr(servConn.conn.ConnectTo(servConn.cfg.Host))
}

func (servConn *serverConnection) join(channel string) {
	servConn.createChatBox(channel, CHATBOX_CHANNEL)
	servConn.conn.Join(channel)
}

func (servConn *serverConnection) part(channel, reason string) {
	if channel[0] == '#' {
		servConn.conn.Part(channel, reason)
	}
	cb := servConn.getChatBox(channel)
	if cb == nil {
		log.Panicln("user not on channel:", channel)
	}
	servConn.deleteChatBox(channel)
}

func (servConn *serverConnection) getChatBox(id string) *chatBox {
	for _, cb := range servConn.chatBoxes {
		if cb.id == id {
			return cb
		}
	}
	return nil
}

func (servConn *serverConnection) createChatBox(id string, boxType int) *chatBox {
	cb := newChatBox(servConn, id, boxType)
	servConn.chatBoxes = append(servConn.chatBoxes, cb)
	return cb
}

func (servConn *serverConnection) deleteChatBox(id string) {
	for i, cb := range servConn.chatBoxes {
		if cb.id == id {
			cb.close()
			servConn.chatBoxes = append(servConn.chatBoxes[0:i], servConn.chatBoxes[i+1:]...)
			return
		}
	}
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
		chatBoxes: []*chatBox{},
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		for _, channel := range cfg.Autojoin {
			servConn.join(channel)
		}
	})

	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		boxType := CHATBOX_CHANNEL
		if channel == servConn.cfg.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		cb.printMessage(fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), l.Nick, l.Args[1]))
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		boxType := CHATBOX_CHANNEL
		if channel == servConn.cfg.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		cb.printMessage(fmt.Sprintf("%s * %s %s", time.Now().Format("15:04"), l.Nick, l.Args[1]))
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		boxType := CHATBOX_CHANNEL
		if channel == servConn.cfg.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		if (channel == "AUTH" || channel == "*") && servConn.cfg.Nick != channel {
			// servers commonly send these NOTICEs when connecting:
			//
			// :irc.example.org NOTICE AUTH :*** Looking up your hostname...
			// :irc.example.org NOTICE AUTH :*** Found your hostname
			//
			// dropping these messages for now...
			return
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		cb.printMessage(fmt.Sprintf("%s *** %s: %s", time.Now().Format("15:04"), l.Nick, l.Args[1]))
	})

	type nickUpdate struct {
		chat        *chatBox
		Add, Remove []string
		Replace     [][]string
	}
	nickUpdateCh := make(chan *nickUpdate)
	go func() {
		for {
			u := <-nickUpdateCh
			for _, repl := range u.Replace {
				u.chat.nickList.Replace(repl[0], repl[1])
			}
			for _, rm := range u.Remove {
				u.chat.nickList.Remove(rm)
			}
			for _, add := range u.Add {
				u.chat.nickList.Add(add)
			}
			u.chat.updateNickList()
		}
	}()

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[2]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got 353 but user not on channel:", l.Args[2])
			return
		}
		nicks := strings.Split(l.Args[3], " ")
		for i, n := range nicks {
			if n == "" {
				nicks = append(nicks[0:i], nicks[i+1:]...)
			}
		}
		nickUpdateCh <- &nickUpdate{chat: cb, Add: nicks}
	})

	conn.HandleFunc(goirc.JOIN, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got JOIN but user not on channel:", l.Args[0])
			return
		}
		if !cb.nickList.Has(l.Nick) {
			nickUpdateCh <- &nickUpdate{chat: cb, Add: []string{l.Nick}}
			cb.printMessage("* " + l.Nick + " has joined " + l.Args[0])
		}
	})

	conn.HandleFunc(goirc.PART, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got PART but user not on channel:", l.Args[0])
			return
		}
		nickUpdateCh <- &nickUpdate{chat: cb, Remove: []string{l.Nick}}
		cb.printMessage("** " + l.Nick + " has left " + l.Args[0])
	})

	conn.HandleFunc(goirc.QUIT, func(c *goirc.Conn, l *goirc.Line) {
		for _, cb := range servConn.chatBoxes {
			if cb.nickList.Has(l.Nick) {
				cb.printMessage("** " + l.Nick + " has quit: " + l.Args[0])
				nickUpdateCh <- &nickUpdate{chat: cb, Remove: []string{l.Nick}}
			}
		}
	})

	conn.HandleFunc(goirc.NICK, func(c *goirc.Conn, l *goirc.Line) {
		if l.Nick == servConn.cfg.Nick {
			servConn.cfg.Nick = l.Args[0]
		}
		for _, cb := range servConn.chatBoxes {
			if cb.nickList.Has(l.Nick) {
				cb.printMessage("** " + l.Nick + " is now known as " + l.Args[0])
				nickUpdateCh <- &nickUpdate{chat: cb, Replace: [][]string{[]string{l.Nick, l.Args[0]}}}
			}
		}
	})

	conn.HandleFunc(goirc.MODE, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		mode := l.Args[1]
		nicks := l.Args[2:]

		log.Printf("op: %#v channel: %#v mode: %#v nicks: %#v", op, channel, mode, nicks)

		if channel[0] == '#' {
			cb := servConn.getChatBox(channel)
			if cb == nil {
				log.Println("got MODE but user not on channel:", channel)
				return
			}
			if len(nicks) == 0 {
				cb.printMessage(fmt.Sprintf("** %s sets mode %s %s", op, mode, channel))
				return
			}
			var add bool
			var idx int
			prefixUpdater := func(symbol string) {
				n := nicks[idx]
				p := cb.nickList.GetPrefix(n)
				if add {
					p += symbol
				} else {
					p = strings.Replace(p, symbol, "", -1)
				}
				cb.nickList.SetPrefix(n, p)
				cb.updateNickList()
				idx++
			}
			for _, b := range mode {
				switch b {
				case '+':
					add = true
				case '-':
					add = false
				case 'q':
					prefixUpdater("~")
				case 'a':
					prefixUpdater("&")
				case 'o':
					prefixUpdater("@")
				case 'h':
					prefixUpdater("%")
				case 'v':
					prefixUpdater("+")
				default:
					panic("unhandled mode modifer:" + string(b))
				}
			}
			cb.printMessage(fmt.Sprintf("** %s sets mode %s %s", op, mode, nicks))
		} else if op == "" {
			nick := channel
			for _, cb := range servConn.chatBoxes {
				if cb.nickList.Has(nick) {
					cb.printMessage(fmt.Sprintf("** %s sets mode %s", nick, mode))
				}
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
