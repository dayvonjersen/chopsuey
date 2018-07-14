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
	conn *goirc.Conn

	retryConnectEnabled bool
	cancelRetryConnect  chan struct{}

	isupport map[string]string
	ip       net.IP
}

func connect(servConn *serverConnection, servState *serverState) (success bool) {
	Println(CLIENT_MESSAGE, servState.AllTabs(), "connecting to:", serverAddr(servState.hostname, servState.port), "...")
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
			Println(CLIENT_ERROR, servState.AllTabs(),
				fmt.Sprintf("couldn't connect to %s after %d retries.",
					serverAddr(servState.hostname, servState.port), CONNECT_RETRIES),
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
	// goirc config
	ident := "chopsuey"
	name := "github.com/generaltso/chopsuey"
	cfg := goirc.NewConfig(servState.user.nick, ident, name)
	cfg.Version = clientState.cfg.Version
	cfg.QuitMessage = clientState.cfg.QuitMessage
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

	// return value
	servConn := &serverConnection{
		conn:                conn,
		retryConnectEnabled: true,
		isupport:            map[string]string{},
	}

	// goirc events
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
		dest := []tabWithInput{getCurrentTabForServer(servState)}
		if dest[0].Index() != servState.tab.Index() {
			dest = append(dest, servState.tab)
		}

		Println(SERVER_MESSAGE, dest, append([]string{l.Cmd}, l.Args[1:]...)...)
	}

	printChannelMessage := func(c *goirc.Conn, l *goirc.Line) {
		// FIXME(COLOURIZE)
		// send to current tab if current tab != channel
		// stub
	}

	printErrorMessage := func(c *goirc.Conn, l *goirc.Line) {
		dest := []tabWithInput{getCurrentTabForServer(servState)}
		if dest[0].Index() != servState.tab.Index() {
			dest = append(dest, servState.tab)
		}

		Println(SERVER_ERROR, dest, append([]string{l.Cmd}, l.Args[1:]...)...)
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

	// MYINFO
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

	// RPL_...
	for _, code := range []string{
		// YOURHOST CREATED BOUNCE UMODEIS
		"002", "003", "010", "221",
		// LUSERCLIENT LUSEROP LUSERUNKNOWN LUSERCHANNELS LUSERME
		"251", "252", "253", "254", "255",
		// ADMINME ADMINLOC1 ADMINLOC2 ADMINEMAIL
		"256", "257", "258", "259",
		// TRYAGAIN
		"263",
		// LOCALUSERS GLOBALUSERS
		"265", "266",
		// NONE
		"300",
		// AWAY USERHOST ISON UNAWAY NOAWAY
		"301", "302", "303", "305", "306",
		// WHOISCERTFP WHOISUSER WHOISSERVER WHOISOPERATOR WHOWASUSER
		"276", "311", "312", "313", "314",
		// WHOISIDLE ENDOFWHOIS WHOISCHANNELS
		"317", "318", "319",
		// VERSION
		"351",
		// MOTDSTART MOTD ENDOFMOTD
		"375", "372", "376",
		// WHOWAS
		"369",
		// YOUREOPER REHASHING
		"381", "382",
	} {
		conn.HandleFunc(code, printServerMessage)
	}

	// RPL_...
	for _, code := range []string{
		// CHANNELMODEIS NOTOPIC TOPICWHOTIME
		"324", "331", "333",
		// INVITING INVITELIST EXCEPTLIST BANLIST
		"341", "346", "348", "367",
		// NOTE(tso): idk if I want to display these end of list messages
		//            same with end of whois/motd
		//            I understand why they exist but idk.
		// -tso 7/12/2018 4:14:48 AM
		// ENDOFNAMES ENDOFBANLIST ENDOFINVITELIST/ENDOFEXCEPTLIST
		"366", "368", "349",
	} {
		conn.HandleFunc(code, printChannelMessage)
	}

	// ERR_...
	for _, code := range []string{"400", "401", "402", "403", "404", "405", "406", "407",
		"408", "409", "411", "412", "413", "414", "415", "421", "422", "423",
		"424", "431", "432", "433", "436", "437", "441", "442", "443", "444",
		"445", "446", "451", "461", "462", "463", "464", "465", "466", "467",
		"471", "472", "473", "474", "475", "476", "477", "478", "481", "482",
		"483", "484", "485", "491", "501", "502", "723"} {
		conn.HandleFunc(code, printErrorMessage)
	}

	getMessageParams := func(l *goirc.Line) (t tabWithInput, nick, msg string) {
		nick = l.Nick
		dest := l.Args[0]
		msg = l.Args[1]

		if dest == servState.user.nick {
			if l.Ident == "service" || isService(nick) {
				return getCurrentTabForServer(servState), nick, msg
			} else {
				pmState := ensurePmState(servConn, servState, nick)
				return pmState.tab, nick, msg
			}
		}
		chanState := ensureChanState(servConn, servState, dest)
		nick = chanState.nickList.Get(nick).String()
		return chanState.tab, nick, msg
	}

	highlighter := func(nick, msg string) bool {
		if servState.user.nick == nick {
			return false
		}

		m, _ := regexp.MatchString(`\b@*`+regexp.QuoteMeta(servState.user.nick)+`(\b|[^\w])`, msg)
		return m
	}

	conn.HandleFunc(goirc.CTCP, func(c *goirc.Conn, l *goirc.Line) {
		// TODO(tso):
		// debugPrint(l)
		if l.Args[0] == "DCC" {
			dccHandler(servConn, l.Nick, l.Args[2])
		}
	})
	conn.HandleFunc(goirc.CTCPREPLY, func(c *goirc.Conn, l *goirc.Line) {
		// TODO(tso):
		// debugPrint(l)
	})

	conn.HandleFunc(goirc.PRIVMSG, func(c *goirc.Conn, l *goirc.Line) {
		t, nick, msg := getMessageParams(l)
		privateMessageWithHighlight(t, highlighter, nick, msg)
	})

	conn.HandleFunc(goirc.ACTION, func(c *goirc.Conn, l *goirc.Line) {
		t, nick, msg := getMessageParams(l)
		nick = strings.Trim(nick, "~&@%+")
		actionMessageWithHighlight(t, highlighter, nick, msg)
	})

	conn.HandleFunc(goirc.NOTICE, func(c *goirc.Conn, l *goirc.Line) {
		var tab tabWithInput = servState.tab
		if isChannel(l.Args[0]) {
			chanState := ensureChanState(servConn, servState, l.Args[0])
			tab = chanState.tab
		} else if l.Args[0] == servState.user.nick {
			tab = getCurrentTabForServer(servState)
		} else if l.Host == l.Src {
			tab = getCurrentTabForServer(servState)
			l.Nick = servState.networkName
		} else {
			log.Println("********************* unhandled NOTICE:")
			debugPrint(l)
		}

		noticeMessageWithHighlight(tab, highlighter, append([]string{l.Nick}, l.Args...)...)
	})

	// NAMREPLY
	conn.HandleFunc("353", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[2]
		chanState := ensureChanState(servConn, servState, channel)
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
			ensureChanState(servConn, servState, channel)
			return
		}
		if !chanState.nickList.Has(l.Nick) {
			chanState.nickList.Add(l.Nick)
			chanState.tab.updateNickList(chanState)
			joinpartMessage(chanState.tab, "->", l.Nick, "has joined", l.Args[0])
		}
	})

	conn.HandleFunc(goirc.PART, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got PART but user not on channel:")
			debugPrint(l)
			return
		}
		chanState.nickList.Remove(l.Nick)
		chanState.tab.updateNickList(chanState)
		msg := []string{"<-", l.Nick, "has left", l.Args[0]}
		if len(l.Args) > 1 {
			msg = append(msg, " ("+l.Args[1]+")")
		}
		joinpartMessage(chanState.tab, msg...)
	})

	conn.HandleFunc(goirc.QUIT, func(c *goirc.Conn, l *goirc.Line) {
		reason := l.Args[0]
		if strings.HasPrefix(reason, "Quit:") {
			reason = strings.TrimPrefix(reason, "Quit:")
		}
		reason = strings.TrimSpace(reason)
		msg := []string{"<-", l.Nick, "has quit"}
		if reason != "" {
			msg = append(msg, "("+reason+")")
		}
		dest := []tabWithInput{}
		for _, chanState := range servState.channels {
			if chanState.nickList.Has(l.Nick) {
				chanState.nickList.Remove(l.Nick)
				chanState.tab.updateNickList(chanState)
				dest = append(dest, chanState.tab)
			}
		}
		Println(JOINPART_MESSAGE, T(dest...), msg...)
	})

	conn.HandleFunc(goirc.KICK, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		who := l.Args[1]
		reason := l.Args[2]

		chanState, ok := servState.channels[channel]
		if !ok {
			log.Println("got KICK but user not on channel:", channel)
			debugPrint(l)
			return
		}

		if who == servState.user.nick {
			msg := fmt.Sprintf(" *** You have been kicked by %s", op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			updateMessage(chanState.tab, msg)
			chanState.nickList = newNickList()
			chanState.tab.updateNickList(chanState)
		} else {
			msg := fmt.Sprintf(" *** %s has been kicked by %s", who, op)
			if reason != op && reason != who {
				msg += ": " + reason
			}
			updateMessage(chanState.tab, msg)
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
				msg := " ** " + oldNick.name + " is now known as " + newNick.name
				updateMessage(chanState.tab, msg)
			}
		}
	})

	conn.HandleFunc(goirc.MODE, func(c *goirc.Conn, l *goirc.Line) {
		op := l.Nick
		channel := l.Args[0]
		mode := l.Args[1]
		nicks := l.Args[2:]

		if isChannel(channel) {
			chanState, ok := servState.channels[channel]
			if !ok {
				log.Println("got MODE but user not on channel:", channel)
				debugPrint(l)
				return
			}
			if op == "" {
				op = servState.networkName
			}
			if len(nicks) == 0 {
				msg := fmt.Sprintf(" ** %s sets mode %s %s", op, mode, channel)
				updateMessage(chanState.tab, msg)
				return
			}

			nickStr := fmt.Sprintf("%s", nicks)
			nickStr = nickStr[1 : len(nickStr)-1]
			msg := fmt.Sprintf(" ** %s sets mode %s %s", op, mode, nickStr)
			updateMessage(chanState.tab, msg)

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
					msg := fmt.Sprintf(" ** %s sets mode %s", nick, mode)
					updateMessage(chanState.tab, msg)
				}
			}
		}
	})

	// TOPIC
	conn.HandleFunc("332", func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[1]
		topic := l.Args[2]

		chanState := ensureChanState(servConn, servState, channel)
		chanState.topic = stripFmtChars(topic)
		chanState.tab.Update(servState, chanState)
		updateMessage(chanState.tab, "topic for", channel, "is", topic)
	})

	conn.HandleFunc(goirc.TOPIC, func(c *goirc.Conn, l *goirc.Line) {
		channel := l.Args[0]
		topic := l.Args[1]
		who := l.Src

		if i := strings.Index(who, "!"); i != -1 {
			who = who[0:i]
		}

		chanState := ensureChanState(servConn, servState, channel)
		chanState.topic = stripFmtChars(topic)
		chanState.tab.Update(servState, chanState)
		updateMessage(chanState.tab, who, "has changed the topic for", channel, "to", topic)
	})

	// LISTSTART
	conn.HandleFunc("321", func(c *goirc.Conn, l *goirc.Line) {
		if servState.channelList == nil {
			servState.channelList = NewChannelList(servConn, servState)
		}
		servState.channelList.inProgress = true
	})

	// LIST
	conn.HandleFunc("322", func(c *goirc.Conn, l *goirc.Line) {
		if servState.channelList == nil {
			return
		}

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

		// NOTE(tso): some networks, such as snoonet put +s channels in the LIST
		//            but hide the channel name by putting * in the channel field.
		// -tso 7/12/2018 3:52:30 AM
		if !isChannel(channel) {
			return
		}
		users, err := strconv.Atoi(args[4])
		if err != nil {
			// this caught the problem before so I'm keeping it for good luck
			checkErr(err)
			debugPrint(l)
		}
		topic := stripFmtChars(strings.TrimSpace(args[5][1:]))

		servState.channelList.mu.Lock()
		defer servState.channelList.mu.Unlock()
		servState.channelList.Add(channel, users, topic)
	})

	// LISTEND
	conn.HandleFunc("323", func(c *goirc.Conn, l *goirc.Line) {
		if servState.channelList == nil {
			return
		}

		servState.channelList.inProgress = false
		servState.channelList.complete = true
	})

	return servConn
}
