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
	goircCfg *goirc.Config
	conn     *goirc.Conn

	retryConnectEnabled bool
	cancelRetryConnect  chan struct{}

	isupport map[string]string
	IP       net.IP
}

func (servConn *serverConnection) Connect(servState *serverState) {
	cancel := make(chan struct{})
	if servConn.retryConnectEnabled {
		go func() {
			for i := 0; i < MAX_CONNECT_RETRIES; i++ {
				select {
				case <-cancel:
					return
				default:
					servState.connState = CONNECTING
					servState.tab.Update(servState)

					err := servConn.conn.ConnectTo(servState.hostname)
					if err != nil {
						servState.connState = CONNECTION_ERROR
						servState.lastError = err
						servState.tab.Update(servState)
					} else {
						servState.connState = CONNECTION_START
						servState.tab.Update(servState)
						return
					}
				}
			}
		}()
	} else {
		err := servConn.conn.ConnectTo(servState.hostname)
		if err != nil {
			servState.connState = CONNECTION_ERROR
			servState.lastError = err
			servState.tab.Update(servState)
		}
	}
	servConn.cancelRetryConnect = cancel
}

func (servConn *serverConnection) Part(channel, reason string, servState *serverState) {
	if channel[0] == '#' {
		servConn.conn.Part(channel, reason)
	}
	chanState, ok := servState.channels[channel]
	if !ok {
		log.Panicln("user not on channel:", channel)
	}
	chanState.tab.Close()
	delete(servState.channels, chanState.channel)
}

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
		conn:                conn,
		retryConnectEnabled: true,
		goircCfg:            goircCfg,
		isupport:            map[string]string{},
	}

	conn.HandleFunc(goirc.CONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		servState.connState = CONNECTED
		servState.tab.Update(servState)
		connectedCallback()
	})

	conn.HandleFunc(goirc.DISCONNECTED, func(c *goirc.Conn, l *goirc.Line) {
		servState.connState = DISCONNECTED
		servState.tab.Update(servState)

		if servConn.retryConnectEnabled {
			connectedCallback = func() {
				for _, channel := range servState.channels {
					conn.Join(channel.channel)
				}
			}
			servConn.Connect(servState)
		}
	})

	printServerMessage := func(c *goirc.Conn, l *goirc.Line) {
		str := color(now(), LightGray) + " " + color(l.Cmd+": "+strings.Join(l.Args[1:], " "), Blue)
		servState.tab.Println(str)
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
		servState.tab.Println(color(now(), LightGray) + " " + color("ERROR("+l.Cmd+")", White, Red) + ": " + color(strings.Join(l.Args[1:], " "), Red))
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
			if l.Ident == "service" {
				tab = servState.tab
			} else {
				pmState, ok := servState.privmsgs[l.Nick]
				if !ok {
					pmState = &privmsgState{
						nick: l.Nick,
					}
					pmState.tab = NewPrivmsgTab(servConn, servState, pmState)
				}
				tab = pmState.tab
			}
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
			}
			tab = chanState.tab
		}
		tab.Println(fmt.Sprintf(fmtstr, now(), nick, l.Args[1]))
	}
	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		printer("PRIVMSG", color("%s", LightGrey)+" "+color("%s", DarkGrey)+" %s", l)
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		printer("ACTION", color("%s", LightGrey)+color(" *%s %s*", DarkGrey), l)
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		if l.Host == l.Src {
			// servers commonly send these NOTICEs when connecting:
			//
			// :irc.example.org NOTICE AUTH :*** Looking up your hostname...
			// :irc.example.org NOTICE AUTH :*** Found your hostname
			//
			printServerMessage(c, l)
			return
		}
		printer("NOTICE", color("%s *** %s: %s", Orange), l)
	})

	ensureChanState := func(channel string) *channelState {
		chanState, ok := servState.channels[channel]
		if !ok {
			chanState = &channelState{
				channel:  channel,
				nickList: newNickList(),
			}
			chanState.tab = NewChannelTab(servConn, servState, chanState)
			servState.channels[channel] = chanState
		}
		return chanState
	}

	// NAMES
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[2]
		chanState := ensureChanState(channel)
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
			conn.Join(channel)
			ensureChanState(channel)
			return
		}
		if !chanState.nickList.Has(l.Nick) {
			chanState.nickList.Add(l.Nick)
			chanState.tab.updateNickList(chanState)
			if !clientCfg.HideJoinParts {
				chanState.tab.Println(color(now(), LightGrey) + italic(color(" -> "+l.Nick+" has joined "+l.Args[0], Orange)))
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
			msg := " <- " + l.Nick + " has left " + l.Args[0]
			if len(l.Args) > 1 {
				msg += " (" + l.Args[1] + ")"
			}
			chanState.tab.Println(color(now(), LightGrey) + italic(color(msg, Orange)))
		}
	})

	conn.HandleFunc(goirc.QUIT, func(c *goirc.Conn, l *goirc.Line) {
		reason := l.Args[0]
		if strings.HasPrefix(reason, "Quit:") {
			reason = strings.TrimPrefix(reason, "Quit:")
		}
		reason = strings.TrimSpace(reason)
		msg := " <- " + l.Nick + " has quit"
		if reason != "" {
			msg += ": " + reason
		}
		for _, chanState := range servState.channels {
			if chanState.nickList.Has(l.Nick) {
				chanState.nickList.Remove(l.Nick)
				chanState.tab.updateNickList(chanState)
				if !clientCfg.HideJoinParts {
					chanState.tab.Println(color(now(), LightGrey) + italic(color(msg, Orange)))
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
			msg := fmt.Sprintf(" *** You have been kicked by %s", op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			chanState.tab.Println(color(now(), LightGrey) + color(msg, Red))
			chanState.nickList = newNickList()
			chanState.tab.updateNickList(chanState)
		} else {
			msg := fmt.Sprintf(" *** %s has been kicked by %s", who, op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			chanState.tab.Println(color(now(), LightGrey) + color(msg, Orange))
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
				chanState.tab.Println(color(now(), LightGrey) + italic(color(" ** "+oldNick.name+" is now known as "+newNick.name, Orange)))
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
				chanState.tab.Println(color(now(), LightGrey) + italic(color(fmt.Sprintf(" ** %s sets mode %s %s", op, mode, channel), Orange)))
				return
			}

			nickStr := fmt.Sprintf("%s", nicks)
			nickStr = nickStr[1 : len(nickStr)-1]
			chanState.tab.Println(color(now(), LightGrey) + italic(color(fmt.Sprintf(" ** %s sets mode %s %s", op, mode, nickStr), Orange)))

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
					chanState.tab.Println(color(now(), LightGrey) + italic(color(fmt.Sprintf(" ** %s sets mode %s", nick, mode), Orange)))
				}
			}
		}
	})

	conn.HandleFunc("332", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		topic := l.Args[2]

		chanState := ensureChanState(channel)
		chanState.topic = topic
		// NOTE(tso): probably should put this in Update() but fuck it
		chanState.tab.topicInput.SetText(topic)
		chanState.tab.Println(color(now(), LightGrey) + " topic for " + channel + " is " + topic)
	})

	conn.HandleFunc(goirc.TOPIC, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		topic := l.Args[1]
		who := l.Src

		if i := strings.Index(who, "!"); i != -1 {
			who = who[0:i]
		}

		chanState := ensureChanState(channel)
		chanState.topic = topic
		// NOTE(tso): probably should put this in Update() but fuck it
		chanState.tab.topicInput.SetText(topic)
		chanState.tab.Println(color(now(), LightGrey) + fmt.Sprintf(" %s has changed the topic for %s to %s", who, channel, topic))
	})

	// START OF /LIST
	conn.HandleFunc("321", func(c *goirc.Conn, l *goirc.Line) {
		if servState.channelList == nil {
			log.Println("got 321 but servState.channelList is nil")
			return
		}
		servState.channelList.inProgress = true
	})

	// LIST
	conn.HandleFunc("322", func(c *goirc.Conn, l *goirc.Line) {
		args := strings.SplitN(l.Raw, " ", 6)
		/*
			args = []string{
				":irc.example.org",
				"322",
				"nick",
				"#channel",
				"4", // user count
				":[+nt] some topic",
			}
		*/
		channel := args[3]
		users, err := strconv.Atoi(args[4])
		if err != nil {
			// this caught the problem before so I'm keeping it for good luck
			checkErr(err)
			debugPrint(l)
		}
		topic := strings.TrimSpace(args[5][1:])

		if servState.channelList == nil {
			servState.channelList = NewChannelList(servConn, servState)
		}

		servState.channelList.mu.Lock()
		defer servState.channelList.mu.Unlock()
		servState.channelList.Add(channel, users, topic)
	})

	// END OF /LIST
	conn.HandleFunc("323", func(c *goirc.Conn, l *goirc.Line) {
		if servState.channelList == nil {
			log.Println("got 323 but servState.channelList is nil")
			return
		}
		servState.channelList.inProgress = false
		servState.channelList.complete = true
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
