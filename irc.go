package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	goirc "github.com/fluffle/goirc/client"
)

type serverConnection struct {
	cfg  *goirc.Config
	conn *goirc.Conn

	retryConnectEnabled bool
	cancelRetryConnect  chan struct{}

	isupport map[string]string
	IP       net.IP
}

func connect(servConn *serverConnection, servState *serverState) (success bool) {
	servState.connState = CONNECTING
	servState.tab.Update(servState)

	err := servConn.conn.Connect()
	if err != nil {
		servState.connState = CONNECTION_ERROR
		servState.lastError = err
		servState.tab.Update(servState)
		return false
	}
	servState.connState = CONNECTION_START
	servState.tab.Update(servState)
	return true
}

func (servConn *serverConnection) Connect(servState *serverState) {
	cancel := make(chan struct{})
	servConn.cancelRetryConnect = cancel
	if !servConn.retryConnectEnabled {
		connect(servConn, servState)
	} else {
		go func() {
			for i := 0; i < CONNECT_RETRIES; i++ {
				select {
				case <-cancel:
					return
				default:
					if connect(servConn, servState) {
						return
					}
					<-time.After(CONNECT_RETRY_INTERVAL)
				}
			}
			servState.tab.Println(
				// FIXME(COLOURIZE)
				fmt.Sprintf("couldn't connect to %s:%d after %d retries.", servState.hostname, servState.port, CONNECT_RETRIES),
			)
		}()
	}
}

// FIXME(tso): does this need to be a member function?
//             I think we could access servConn.conn directly and then close the tab.
//             wait we wanted to stop closing tabs for /part
//             fix this.
func (servConn *serverConnection) Part(channel, reason string, servState *serverState) {
	if isChannel(channel) {
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
	ident := "chopsuey"
	name := "github.com/generaltso/chopsuey"
	cfg := goirc.NewConfig(servState.user.nick, ident, name)
	cfg.Version = clientCfg.Version
	cfg.QuitMessage = clientCfg.QuitMessage
	if servState.ssl {
		cfg.SSL = true
		cfg.SSLConfig = &tls.Config{
			ServerName:         servState.hostname,
			InsecureSkipVerify: true,
		}
		cfg.NewNick = func(n string) string { return n + "^" }
	}
	cfg.Server = serverAddr(servState.hostname, servState.port)
	conn := goirc.Client(cfg)

	servConn := &serverConnection{
		conn:                conn,
		retryConnectEnabled: true,
		cfg:                 cfg,
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
		// FIXME(COLOURIZE)
		str := color(now(), LightGray) + " " + color(l.Cmd+": "+strings.Join(l.Args[1:], " "), Blue)
		servState.tab.Println(str)
	}

	// WELCOME
	conn.HandleFunc("001", func(c *goirc.Conn, l *goirc.Line) {
		nick := l.Args[0]
		// if nickname is already in use (433)
		// welcome message (001) will tell us what they renamed us to
		if nick != servState.user.nick {
			servState.user.nick = nick
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

	for _, code := range []string{
		"251", "252", "253", "254", "255", // LUSERS
		"375", "372", "376", // MOTD
		"256", "257", "258", "259", // ADMIN
		"311", "312", "313", "317", "318", "319", // WHOIS
		"314", "369", // WHOWAS
	} {
		conn.HandleFunc(code, printServerMessage)
	}

	// "is connecting from"
	// NOTE(tso): I've never seen this message come in...
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
		// FIXME(COLOURIZE)
		servState.tab.Println(color(now(), LightGray) + " " + color("ERROR("+l.Cmd+")", White, Red) + ": " + color(strings.Join(l.Args[1:], " "), Red))
	}

	for _, code := range []string{"401", "402", "403", "404", "405", "406", "407",
		"408", "409", "411", "412", "413", "414", "415", "421", "422", "423",
		"424", "431", "432", "433", "436", "437", "441", "442", "443", "444",
		"445", "446", "451", "461", "462", "463", "464", "465", "466", "467",
		"471", "472", "473", "474", "475", "476", "477", "478", "481", "482",
		"483", "484", "485", "491", "501", "502"} {
		conn.HandleFunc(code, printErrorMessage)
	}

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

	highlighter := func(text string, l *goirc.Line) string {
		// NOTE(tso): not using compiled regexp here because user's nick can change
		//			  unless recompiling a new one when the nick changes will really
		//			  give that much of a performance increase
		// -tso 7/10/2018 6:58:36 AM
		m, _ := regexp.MatchString(`\b@*`+regexp.QuoteMeta(servState.user.nick)+`(\b|[^\w])`, l.Args[1])
		if m {
			// FIXME(COLOURIZE)
			return bold(color(" * ", Black, Yellow)) + text
		}
		return text
	}

	conn.HandleFunc(goirc.CTCP, func(c *goirc.Conn, l *goirc.Line) {
		// debugPrint(l)
		if l.Args[0] == "DCC" {
			dccHandler(servConn, l.Nick, l.Args[2])
		}
	})
	conn.HandleFunc(goirc.CTCPREPLY, func(c *goirc.Conn, l *goirc.Line) {
		// debugPrint(l)
	})

	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		if l.Args[0] != servState.user.nick {
			// FIXME(COLOURIZE)
			printer("PRIVMSG", color("%s", LightGrey)+" "+highlighter(color("%s", DarkGrey), l)+" %s", l)
		} else {
			printer("PRIVMSG", color("%s", LightGrey)+" "+color("%s", DarkGrey)+" %s", l)
		}
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		if l.Args[0] != servState.user.nick {
			// FIXME(COLOURIZE)
			printer("ACTION", color("%s", LightGrey)+highlighter(color(" *%s %s*", DarkGrey), l), l)
		} else {
			printer("ACTION", color("%s", LightGrey)+color(" *%s %s*", DarkGrey), l)
		}
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
		// FIXME(COLOURIZE)
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
				// FIXME(COLOURIZE)
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
			// FIXME(COLOURIZE)
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
					// FIXME(COLOURIZE)
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
			// FIXME(COLOURIZE)
			chanState.tab.Println(color(now(), LightGrey) + color(msg, Red))
			chanState.nickList = newNickList()
			chanState.tab.updateNickList(chanState)
		} else {
			msg := fmt.Sprintf(" *** %s has been kicked by %s", who, op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			// FIXME(COLOURIZE)
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
				// FIXME(COLOURIZE)
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
				// FIXME(COLOURIZE)
				chanState.tab.Println(color(now(), LightGrey) + italic(color(fmt.Sprintf(" ** %s sets mode %s %s", op, mode, channel), Orange)))
				return
			}

			nickStr := fmt.Sprintf("%s", nicks)
			nickStr = nickStr[1 : len(nickStr)-1]
			// FIXME(COLOURIZE)
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
					// FIXME(COLOURIZE)
					chanState.tab.Println(color(now(), LightGrey) + italic(color(fmt.Sprintf(" ** %s sets mode %s", nick, mode), Orange)))
				}
			}
		}
	})

	// TOPIC
	conn.HandleFunc("332", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		topic := l.Args[2]

		chanState := ensureChanState(channel)
		chanState.topic = topic
		chanState.tab.Update(servState, chanState)
		// FIXME(COLOURIZE)
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
		chanState.tab.Update(servState, chanState)
		// FIXME(COLOURIZE)
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
