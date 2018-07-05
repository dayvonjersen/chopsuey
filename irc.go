package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"

	goirc "github.com/fluffle/goirc/client"
)

const MAX_CONNECT_RETRIES = 100

type serverConnection struct {
	IP       net.IP
	hostname string

	goircCfg *goirc.Config
	conn     *goirc.Conn
	// channelList *channelList

	retryConnectEnabled bool

	isupport map[string]string
}

func (servConn *serverConnection) Connect() {
	servConn.conn.ConnectTo(servConn.hostname)
	// go servConn.retryConnect()
}

/*
	cb := servConn.createChatBox(servConn.serverName, CHATBOX_SERVER)
}
*/
func (servConn *serverConnection) retryConnect() {
}

/*
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
*/
func (servConn *serverConnection) Join(channel string, servState *serverState) {
	chanState, ok := servState.channels[channel]
	if !ok {
		// open a new tab
		chanState = &channelState{
			channel:  channel,
			nickList: newNickList(),
		}
		chanState.tab = NewChannelTab(servConn, servState, chanState)
		servState.channels[channel] = chanState
	}
	servConn.conn.Join(channel)
}

/*
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
*/
func NewServerConnection(servState *serverState, connectedCallback func()) *serverConnection {
	goircCfg := goirc.NewConfig(servState.user.nick)
	goircCfg.Version = clientCfg.Version
	if servState.ssl {
		goircCfg.SSL = true
		goircCfg.SSLConfig = &tls.Config{
			ServerName:         servState.hostname,
			InsecureSkipVerify: true,
		}
		goircCfg.NewNick = func(n string) string { return n + "^" }
	}
	goircCfg.Server = fmt.Sprintf("%s:%d", servState.hostname, servState.port)
	conn := goirc.Client(goircCfg)

	servConn := &serverConnection{
		hostname:            servState.hostname,
		conn:                conn,
		retryConnectEnabled: true,
		goircCfg:            goircCfg,
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		servState.connected = true
		servState.tab.Update(servState)
		connectedCallback()
	})

	conn.HandleFunc(goirc.DISCONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		servState.connected = false
		servState.tab.Update(servState)

		if servConn.retryConnectEnabled {
			go servConn.retryConnect()
		}
	})

	printServerMessage := func(c *goirc.Conn, l *goirc.Line) {
		servState.tab.Println(now() + " " + l.Cmd + ": " + strings.Join(l.Args[1:], " "))
	}

	// WELCOME
	conn.HandleFunc("001", func(c *goirc.Conn, l *goirc.Line) {
		// if nickname is already in use (433)
		// welcome message (001) will tell us what they renamed us to
		if l.Args[0] != servState.user.nick {
			servState.user.nick = l.Args[0]
			servState.tab.Update(servState)
		}
		printServerMessage(c, l)
	})
	conn.HandleFunc("002", printServerMessage)
	conn.HandleFunc("003", printServerMessage)
	conn.HandleFunc("004", func(c *goirc.Conn, l *goirc.Line) {
		servState.networkName = l.Args[1]
		servState.tab.Update(servState)
		printServerMessage(c, l)
	})

	// ISUPPORT
	conn.HandleFunc("005", func(c *goirc.Conn, l *goirc.Line) {
		// l.Args[0] is nick
		// l.Args[-1] is "are supported by this server"
		args := l.Args[1 : len(l.Args)-1]
		for _, st := range args {
			s := strings.Split(st, "=")
			k := s[0]
			v := ""
			if len(s) > 1 {
				v = s[1]
			}
			servConn.isupport[k] = v
			if k == "NETWORK" {
				if servState.networkName != v {
					servState.networkName = v
					servState.tab.Update(servState)
				}
			}
		}
		printServerMessage(c, l)
	})

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
		if len(l.Args) == 3 && l.Args[1] == servState.user.nick {
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
		servState.tab.Println(now() + " " + l.Cmd + " ERROR: " + strings.Join(l.Args[1:], " "))
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
		// debugPrint(l)
		if l.Args[0] == "DCC" {
			dccHandler(servConn, l.Nick, l.Args[2])
		}
	})
	conn.HandleFunc(goirc.CTCPREPLY, func(c *goirc.Conn, l *goirc.Line) {
		// debugPrint(l)
	})

	printer := func(code, fmtstr string, l *goirc.Line) {
		var (
			tab  tabViewWithInput
			nick string
		)
		if l.Args[0] == servState.user.nick {
			nick = l.Nick

			// NOTE(tso): inline NOTICEs from services etc here.
			pmState, ok := servState.privmsgs[l.Nick]
			if !ok {
				pmState = &privmsgState{
					nick: l.Nick,
				}
				pmState.tab = NewPrivmsgTab(servConn, servState, pmState)
				servState.privmsgs[l.Nick] = pmState
			}
			tab = pmState.tab
		} else {
			chanState, ok := servState.channels[l.Args[0]]
			if ok {
				nick = chanState.nickList.Get(l.Nick).String()
			} else {
				log.Println("got", code, " but user not on channel")
				chanState = &channelState{
					channel: l.Args[0],
				}
				chanState.tab = NewChannelTab(servConn, servState, chanState)
				servState.channels[l.Args[0]] = chanState
			}
			tab = chanState.tab
		}
		tab.Println(fmt.Sprintf(fmtstr, now(), nick, l.Args[1]))
	}
	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		printer("PRIVMSG", "%s <%s> %s", l)
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		printer("ACTION", "%s * %s %s", l)
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		// debugPrint(l)
		channel := strings.TrimSpace(l.Args[0])
		if (channel == "AUTH" || channel == "*" || channel == "") && servState.user.nick != channel {
			// servers commonly send these NOTICEs when connecting:
			//
			// :irc.example.org NOTICE AUTH :*** Looking up your hostname...
			// :irc.example.org NOTICE AUTH :*** Found your hostname
			//
			printServerMessage(c, l)
			return
		}
		printer("NOTICE", "%s *** %s: %s", l)
	})

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[2]
		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got 353 but user not on channel:", l.Args[2])
			return
		}
		nicks := strings.Split(l.Args[3], " ")
		for _, n := range nicks {
			if n != "" {
				chanState.nickList.Add(n)
			}
		}
		chanState.tab.updateNickList(chanState)
	})

	conn.HandleFunc(goirc.JOIN, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		chanState, ok := servState.channels[channel]
		if !ok {
			// forced join
			servConn.Join(channel, servState)
			return
		}
		if !chanState.nickList.Has(l.Nick) {
			chanState.nickList.Add(l.Nick)
			chanState.tab.updateNickList(chanState)
			if !clientCfg.HideJoinParts {
				chanState.tab.Println(now() + " -> " + l.Nick + " has joined " + l.Args[0])
			}
		}
	})

	conn.HandleFunc(goirc.PART, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got PART but user not on channel:", l.Args[0])
			return
		}
		chanState.nickList.Remove(l.Nick)
		chanState.tab.updateNickList(chanState)
		if !clientCfg.HideJoinParts {
			msg := now() + " <- " + l.Nick + " has left " + l.Args[0]
			if len(l.Args) > 1 {
				msg += " (" + l.Args[1] + ")"
			}
			chanState.tab.Println(msg)
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
		for _, chanState := range servState.channels {
			if chanState.nickList.Has(l.Nick) {
				chanState.nickList.Remove(l.Nick)
				chanState.tab.updateNickList(chanState)
				if !clientCfg.HideJoinParts {
					chanState.tab.Println(msg)
				}
			}
		}
	})

	conn.HandleFunc(goirc.KICK, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		who := l.Args[1]
		reason := l.Args[2]

		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got KICK but user not on channel:", channel)
			return
		}

		if who == servState.user.nick {
			msg := fmt.Sprintf("%s *** You have been kicked by %s", now(), op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			chanState.tab.Println(msg)
			chanState.nickList = newNickList()
			chanState.tab.updateNickList(chanState)
		} else {
			msg := fmt.Sprintf("%s *** %s has been kicked by %s", now(), who, op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			chanState.tab.Println(msg)
			chanState.nickList.Remove(who)
			chanState.tab.updateNickList(chanState)
		}
	})

	conn.HandleFunc(goirc.NICK, func(c *goirc.Conn, l *goirc.Line) {
		oldNick := newNick(l.Nick)
		newNick := newNick(l.Args[0])
		if oldNick.name == servState.user.nick {
			servState.user.nick = newNick.name
			servState.tab.Update(servState)
		}
		for _, chanState := range servState.channels {
			if chanState.nickList.Has(oldNick.name) {
				newNick.prefix = oldNick.prefix
				chanState.nickList.Set(oldNick.name, newNick)
				chanState.tab.updateNickList(chanState)
				chanState.tab.Println(now() + " ** " + oldNick.name + " is now known as " + newNick.name)
			}
		}
	})

	conn.HandleFunc(goirc.MODE, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		mode := l.Args[1]
		nicks := l.Args[2:]

		if channel[0] == '#' {
			chanState, ok := servState.channels[channel]
			if !ok {
				log.Println("got MODE but user not on channel:", channel)
				return
			}
			if op == "" {
				op = servState.networkName
			}
			if len(nicks) == 0 {
				chanState.tab.Println(fmt.Sprintf("%s ** %s sets mode %s %s", now(), op, mode, channel))
				return
			}

			nickStr := fmt.Sprintf("%s", nicks)
			nickStr = nickStr[1 : len(nickStr)-1]
			chanState.tab.Println(fmt.Sprintf("%s ** %s sets mode %s %s", now(), op, mode, nickStr))

			var add bool
			var idx int
			prefixUpdater := func(symbol string) {
				n := nicks[idx]
				nick := chanState.nickList.Get(n)
				if add {
					nick.prefix += symbol
				} else {
					nick.prefix = strings.Replace(nick.prefix, symbol, "", -1)
				}
				chanState.nickList.Set(n, nick)
				chanState.tab.updateNickList(chanState)
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
			for _, chanState := range servState.channels {
				if chanState.nickList.Has(nick) || nick == servState.user.nick {
					chanState.tab.Println(fmt.Sprintf("%s ** %s sets mode %s", now(), nick, mode))
				}
			}
		}
	})

	conn.HandleFunc("332", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		topic := l.Args[2]

		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got TOPIC but user not on channel:", channel)
			return
		}
		chanState.topic = topic
		// NOTE(tso): probably should put this in Update() but fuck it
		chanState.tab.topicInput.SetText(topic)
		chanState.tab.Println(fmt.Sprintf("*** topic for %s is %s", channel, topic))
	})

	conn.HandleFunc(goirc.TOPIC, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		topic := l.Args[1]
		who := l.Src

		if i := strings.Index(who, "!"); i != -1 {
			who = who[0:i]
		}

		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got TOPIC but user not on channel:", channel)
			return
		}
		chanState.topic = topic
		// NOTE(tso): probably should put this in Update() but fuck it
		chanState.tab.topicInput.SetText(topic)
		chanState.tab.Println(fmt.Sprintf("%s *** %s has changed the topic for %s to %s", now(), who, channel, topic))
	})
	/*
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
	*/
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
