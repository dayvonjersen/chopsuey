package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	goirc "github.com/fluffle/goirc/client"
)

const MAX_CONNECT_RETRIES = 100

type serverConnection struct {
	networkName, Nick string
	IP                net.IP

	cfg          *connectionConfig
	conn         *goirc.Conn
	chatBoxes    []*chatBox
	newChats     chan string
	newChatBoxes chan *chatBox
	closeChats   chan *chatBox
	channelList  *channelList

	retryConnectEnabled bool
}

func (servConn *serverConnection) connect() {
	cb := servConn.createChatBox(servConn.networkName, CHATBOX_SERVER)
	go servConn.retryConnect(cb)
}

func (servConn *serverConnection) retryConnect(cb *chatBox) {
	for i := 0; i < MAX_CONNECT_RETRIES; i++ {
		cb.printMessage(now() + " connecting to " + servConn.cfg.ServerString() + "...")
		statusBar.SetText("connecting to " + servConn.cfg.ServerString() + "...")
		err := servConn.conn.ConnectTo(servConn.cfg.Host)
		if err != nil {
			cb.printMessage(now() + " " + err.Error())
			statusBar.SetText("couldn't connect to " + servConn.cfg.ServerString())
			if !servConn.retryConnectEnabled {
				return
			}
		} else {
			statusBar.SetText(servConn.Nick + " connected to " + servConn.networkName)
			break
		}
	}
}

func (servConn *serverConnection) join(channel string) {
	cb := servConn.getChatBox(channel)
	if cb == nil {
		servConn.createChatBox(channel, CHATBOX_CHANNEL)
	}
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

func newServerConnection(cfg *connectionConfig) *serverConnection {
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
		Nick:                cfg.Nick,
		networkName:         cfg.ServerString(),
		cfg:                 cfg,
		conn:                conn,
		chatBoxes:           []*chatBox{},
		retryConnectEnabled: true,
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		for _, channel := range cfg.AutoJoin {
			servConn.join(channel)
		}
	})

	conn.HandleFunc(goirc.DISCONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		cb := servConn.getChatBox(servConn.networkName)
		if cb == nil {
			log.Println("chatbox for server tab not found:", servConn.networkName)
			return
		}
		cb.printMessage(now() + " disconnected x_x")

		statusBar.SetText("disconnected x_x")

		if servConn.retryConnectEnabled {
			go servConn.retryConnect(cb)
		}
	})

	printServerMessage := func(c *goirc.Conn, l *goirc.Line) {
		cb := servConn.getChatBox(servConn.networkName)
		if cb == nil {
			log.Println("chatbox for server tab not found:", servConn.networkName)
			return
		}
		cb.printMessage(now() + " " + l.Cmd + ": " + strings.Join(l.Args[1:], " "))
	}

	// WELCOME
	conn.HandleFunc("001", printServerMessage)
	conn.HandleFunc("002", printServerMessage)
	conn.HandleFunc("003", printServerMessage)
	conn.HandleFunc("004", func(c *goirc.Conn, l *goirc.Line) {
		cb := servConn.getChatBox(servConn.networkName)
		if cb == nil {
			log.Println("chatbox for server tab not found:", servConn.networkName)
			return
		}
		servConn.networkName = l.Args[1]
		cb.id = servConn.networkName
		mw.WindowBase.Synchronize(func() {
			cb.tabPage.SetTitle(servConn.networkName)
			statusBar.SetText(cfg.Nick + " connected to " + servConn.networkName)
		})
		printServerMessage(c, l)
	})
	// BOUNCE
	conn.HandleFunc("005", printServerMessage)
	// LUSERS
	conn.HandleFunc("251", printServerMessage)
	conn.HandleFunc("252", printServerMessage)
	conn.HandleFunc("253", printServerMessage)
	conn.HandleFunc("254", printServerMessage)
	conn.HandleFunc("255", printServerMessage)
	// MOTD
	conn.HandleFunc("375", printServerMessage)
	conn.HandleFunc("372", printServerMessage)
	conn.HandleFunc("376", printServerMessage)
	// ADMIN
	conn.HandleFunc("256", printServerMessage)
	conn.HandleFunc("257", printServerMessage)
	conn.HandleFunc("258", printServerMessage)
	conn.HandleFunc("259", printServerMessage)
	// WHOIS
	conn.HandleFunc("311", printServerMessage)
	conn.HandleFunc("312", printServerMessage)
	conn.HandleFunc("313", printServerMessage)
	conn.HandleFunc("317", printServerMessage)
	conn.HandleFunc("318", printServerMessage)
	conn.HandleFunc("319", printServerMessage)
	// WHOWAS
	conn.HandleFunc("314", printServerMessage)
	conn.HandleFunc("369", printServerMessage)

	// "is connecting from"
	conn.HandleFunc("378", func(c *goirc.Conn, l *goirc.Line) {
		if len(l.Args) == 3 && l.Args[1] == servConn.Nick {
			s := strings.Split(l.Args[2], " ")
			ipStr := s[len(s)-1]
			ip := net.ParseIP(ipStr)
			if ip != nil {
				if ip4 := ip.To4(); ip4 != nil {
					servConn.IP = ip4
				}
			}
		}
		printServerMessage(c, l)
	})
	// TODO: there are more...

	printErrorMessage := func(c *goirc.Conn, l *goirc.Line) {
		cb := servConn.getChatBox(servConn.networkName)
		if cb == nil {
			log.Println("chatbox for server tab not found:", servConn.networkName)
			return
		}
		cb.printMessage(now() + " " + l.Cmd + " >>> >>> >>> ERROR: " + strings.Join(l.Args[1:], " <<< <<< <<<"))
	}

	conn.HandleFunc("401", printErrorMessage)
	conn.HandleFunc("402", printErrorMessage)
	conn.HandleFunc("403", printErrorMessage)
	conn.HandleFunc("404", printErrorMessage)
	conn.HandleFunc("405", printErrorMessage)
	conn.HandleFunc("406", printErrorMessage)
	conn.HandleFunc("407", printErrorMessage)
	conn.HandleFunc("408", printErrorMessage)
	conn.HandleFunc("409", printErrorMessage)
	conn.HandleFunc("411", printErrorMessage)
	conn.HandleFunc("412", printErrorMessage)
	conn.HandleFunc("413", printErrorMessage)
	conn.HandleFunc("414", printErrorMessage)
	conn.HandleFunc("415", printErrorMessage)
	conn.HandleFunc("421", printErrorMessage)
	conn.HandleFunc("422", printErrorMessage)
	conn.HandleFunc("423", printErrorMessage)
	conn.HandleFunc("424", printErrorMessage)
	conn.HandleFunc("431", printErrorMessage)
	conn.HandleFunc("432", printErrorMessage)
	conn.HandleFunc("433", printErrorMessage)
	conn.HandleFunc("436", printErrorMessage)
	conn.HandleFunc("437", printErrorMessage)
	conn.HandleFunc("441", printErrorMessage)
	conn.HandleFunc("442", printErrorMessage)
	conn.HandleFunc("443", printErrorMessage)
	conn.HandleFunc("444", printErrorMessage)
	conn.HandleFunc("445", printErrorMessage)
	conn.HandleFunc("446", printErrorMessage)
	conn.HandleFunc("451", printErrorMessage)
	conn.HandleFunc("461", printErrorMessage)
	conn.HandleFunc("462", printErrorMessage)
	conn.HandleFunc("463", printErrorMessage)
	conn.HandleFunc("464", printErrorMessage)
	conn.HandleFunc("465", printErrorMessage)
	conn.HandleFunc("466", printErrorMessage)
	conn.HandleFunc("467", printErrorMessage)
	conn.HandleFunc("471", printErrorMessage)
	conn.HandleFunc("472", printErrorMessage)
	conn.HandleFunc("473", printErrorMessage)
	conn.HandleFunc("474", printErrorMessage)
	conn.HandleFunc("475", printErrorMessage)
	conn.HandleFunc("476", printErrorMessage)
	conn.HandleFunc("477", printErrorMessage)
	conn.HandleFunc("478", printErrorMessage)
	conn.HandleFunc("481", printErrorMessage)
	conn.HandleFunc("482", printErrorMessage)
	conn.HandleFunc("483", printErrorMessage)
	conn.HandleFunc("484", printErrorMessage)
	conn.HandleFunc("485", printErrorMessage)
	conn.HandleFunc("491", printErrorMessage)
	conn.HandleFunc("501", printErrorMessage)
	conn.HandleFunc("502", printErrorMessage)
	// I think that's all of them... -_-'

	conn.HandleFunc(goirc.CTCP, func(c *goirc.Conn, l *goirc.Line) {
		debugPrint(l)
		if l.Args[0] == "DCC" {
			dccHandler(servConn, l.Nick, l.Args[2])
		}
	})
	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		boxType := CHATBOX_CHANNEL
		if channel == servConn.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		var nick string
		if boxType == CHATBOX_CHANNEL {
			nick = cb.nickList.Get(l.Nick).String()
		} else {
			nick = channel
		}
		cb.printMessage(fmt.Sprintf("%s <%s> %s", now(), nick, l.Args[1]))
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		boxType := CHATBOX_CHANNEL
		if channel == servConn.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		cb.printMessage(fmt.Sprintf("%s *%s %s*", now(), l.Nick, l.Args[1]))
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		channel := strings.TrimSpace(l.Args[0])
		boxType := CHATBOX_CHANNEL
		if channel == servConn.Nick {
			channel = l.Nick
			boxType = CHATBOX_PRIVMSG
		}
		if (channel == "AUTH" || channel == "*" || channel == "") && servConn.Nick != channel {
			// servers commonly send these NOTICEs when connecting:
			//
			// :irc.example.org NOTICE AUTH :*** Looking up your hostname...
			// :irc.example.org NOTICE AUTH :*** Found your hostname
			//
			printServerMessage(c, l)
			return
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			cb = servConn.createChatBox(channel, boxType)
		}
		cb.printMessage(fmt.Sprintf("%s *** %s: %s", now(), l.Nick, l.Args[1]))
	})

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[2]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got 353 but user not on channel:", l.Args[2])
			return
		}
		nicks := strings.Split(l.Args[3], " ")
		for _, n := range nicks {
			if n != "" {
				cb.nickList.Add(n)
			}
		}
		cb.updateNickList()
	})

	conn.HandleFunc(goirc.JOIN, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			// forced join
			servConn.join(channel)
			return
		}
		if !cb.nickList.Has(l.Nick) {
			cb.nickList.Add(l.Nick)
			cb.updateNickList()
			if !clientCfg.HideJoinParts {
				cb.printMessage(now() + " -> " + l.Nick + " has joined " + l.Args[0])
			}
		}
	})

	conn.HandleFunc(goirc.PART, func(c *goirc.Conn, l *goirc.Line) {
		debugPrint(l)
		channel := l.Args[0]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got PART but user not on channel:", l.Args[0])
			return
		}
		cb.nickList.Remove(l.Nick)
		cb.updateNickList()
		if !clientCfg.HideJoinParts {
			msg := now() + " <- " + l.Nick + " has left " + l.Args[0]
			if len(l.Args) > 1 {
				msg += " (" + l.Args[1] + ")"
			}
			cb.printMessage(msg)
		}
	})

	conn.HandleFunc(goirc.QUIT, func(c *goirc.Conn, l *goirc.Line) {
		reason := l.Args[0]
		if strings.HasPrefix(reason, "Quit:") {
			reason = strings.TrimPrefix(reason, "Quit:")
		}
		reason = strings.TrimSpace(reason)
		msg := now() + " <- " + l.Nick + " has quit"
		if reason != "" {
			msg += ": " + reason
		}
		for _, cb := range servConn.chatBoxes {
			if cb.nickList.Has(l.Nick) {
				cb.nickList.Remove(l.Nick)
				cb.updateNickList()
				if !clientCfg.HideJoinParts {
					cb.printMessage(msg)
				}
			}
		}
	})

	conn.HandleFunc(goirc.KICK, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		who := l.Args[1]
		reason := l.Args[2]

		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got KICK but user not on channel:", channel)
			return
		}

		if who == servConn.Nick {
			msg := fmt.Sprintf("%s *** You have been kicked by %s", now(), op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			cb.printMessage(msg)
			cb.nickList = newNickList()
			cb.updateNickList()
		} else {
			msg := fmt.Sprintf("%s *** %s has been kicked by %s", now(), who, op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			cb.printMessage(msg)
			cb.nickList.Remove(who)
			cb.updateNickList()
		}
	})

	conn.HandleFunc(goirc.NICK, func(c *goirc.Conn, l *goirc.Line) {
		oldNick := newNick(l.Nick)
		newNick := newNick(l.Args[0])
		log.Println(oldNick, newNick, servConn.Nick)
		if oldNick.name == servConn.Nick {
			servConn.Nick = newNick.name
			statusBar.SetText(newNick.name + " connected to " + cfg.ServerString())
		}
		for _, cb := range servConn.chatBoxes {
			if cb.nickList.Has(oldNick.name) {
				newNick.prefix = oldNick.prefix
				cb.nickList.Set(oldNick.name, newNick)
				cb.updateNickList()
				cb.printMessage(now() + " ** " + oldNick.name + " is now known as " + newNick.name)
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
			if op == "" {
				op = servConn.networkName
			}
			if len(nicks) == 0 {
				cb.printMessage(fmt.Sprintf("%s ** %s sets mode %s %s", now(), op, mode, channel))
				return
			}

			nickStr := fmt.Sprintf("%s", nicks)
			nickStr = nickStr[1 : len(nickStr)-1]
			cb.printMessage(fmt.Sprintf("%s ** %s sets mode %s %s", now(), op, mode, nickStr))

			var add bool
			var idx int
			prefixUpdater := func(symbol string) {
				n := nicks[idx]
				nick := cb.nickList.Get(n)
				if add {
					nick.prefix += symbol
				} else {
					nick.prefix = strings.Replace(nick.prefix, symbol, "", -1)
				}
				cb.nickList.Set(n, nick)
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
				case 'b':
					idx++
				default:
					panic("unhandled mode modifer:" + string(b))
				}
			}
		} else if op == "" {
			nick := channel
			for _, cb := range servConn.chatBoxes {
				if cb.nickList.Has(nick) || nick == servConn.Nick {
					cb.printMessage(fmt.Sprintf("%s ** %s sets mode %s", now(), nick, mode))
				}
			}
		}
	})

	conn.HandleFunc("332", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		topic := l.Args[2]
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got TOPIC but user not on channel:", channel)
			return
		}
		cb.topicInput.SetText(topic)
		cb.printMessage(fmt.Sprintf("*** topic for %s is %s", channel, topic))
	})

	conn.HandleFunc(goirc.TOPIC, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		topic := l.Args[1]
		who := l.Src
		if i := strings.Index(who, "!"); i != -1 {
			who = who[0:i]
		}
		cb := servConn.getChatBox(channel)
		if cb == nil {
			log.Println("got TOPIC but user not on channel:", channel)
			return
		}
		cb.topicInput.SetText(topic)
		cb.printMessage(fmt.Sprintf("%s *** %s has changed the topic for %s to %s", now(), who, channel, topic))
	})

	// START OF /LIST
	conn.HandleFunc("321", func(c *goirc.Conn, l *goirc.Line) {
		if servConn.channelList == nil {
			log.Println("got 321 but servConn.channeList is nil")
			return
		}
		servConn.channelList.inProgress = true
	})

	// LIST
	conn.HandleFunc("322", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		users, err := strconv.Atoi(l.Args[2])
		checkErr(err)
		topic := strings.TrimSpace(l.Args[3])

		if servConn.channelList == nil {
			servConn.channelList = newChannelList(servConn)
		}

		servConn.channelList.mu.Lock()
		defer servConn.channelList.mu.Unlock()
		servConn.channelList.Add(channel, users, topic)
	})

	// END OF /LIST
	conn.HandleFunc("323", func(c *goirc.Conn, l *goirc.Line) {
		if servConn.channelList == nil {
			log.Println("got 323 but servConn.channeList is nil")
			return
		}
		servConn.channelList.inProgress = false
		servConn.channelList.complete = true
	})

	return servConn
}

func debugPrint(l *goirc.Line) {
	printf(&goirc.Line{
		Nick:  l.Nick,
		Ident: l.Ident,
		Host:  l.Host,
		Src:   l.Src,
		Cmd:   l.Cmd,
		Raw:   l.Raw,
		Args:  l.Args,
	})
}
