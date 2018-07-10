package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type commandContext struct {
	servConn *serverConnection
	tab      tabViewWithInput

	servState *serverState
	chanState *channelState
	pmState   *privmsgState
}

type clientCommand func(ctx *commandContext, args ...string)

var clientCommands map[string]clientCommand

func init() {
	clientCommands = map[string]clientCommand{
		"clear":  clearCmd,
		"close":  closeCmd,
		"ctcp":   ctcpCmd,
		"join":   joinCmd,
		"kick":   kickCmd,
		"list":   listCmd,
		"me":     meCmd,
		"mode":   modeCmd,
		"msg":    privmsgCmd,
		"nick":   nickCmd,
		"notice": noticeCmd,
		"part":   partCmd,
		"quit":   quitCmd,
		"rejoin": rejoinCmd,
		"server": serverCmd,
		"topic":  topicCmd,

		"connect":    connectCmd,
		"disconnect": disconnectCmd,
		"reconnect":  reconnectCmd,

		"help": helpCmd,
		"exit": exitCmd,

		"raw":  rawCmd,
		"send": sendCmd,
	}
}

// for debug purposes only
func rawCmd(ctx *commandContext, args ...string) {
	ctx.servConn.conn.Raw(strings.Join(args, " "))
}

func helpCmd(ctx *commandContext, args ...string) {
	// TODO(tso): print usage for individual commands
	ctx.tab.Println(`TIPS AND TRICKS:

connect to a network:
/server irc.example.org

to quit the application:
/exit

disconnect from a network without closing all your tabs:
/disconnect

disconnect from a network AND close all associated tabs:
/quit
or on the server connection tab:
/close

leave a channel without closing tab:
/part

leave a channel or privmsg and close tab:
/close
`)
}

func exitCmd(ctx *commandContext, args ...string) {
	os.Exit(0)
}

func sendCmd(ctx *commandContext, args ...string) {
	who := args[0]
	file := "rfc2812.txt"
	fileTransfer(ctx.servConn, who, file)
}

func ctcpCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /ctcp [nick] [message] [args...]")
		return
	}
	ctx.servConn.conn.Ctcp(args[0], args[1], args[2:]...)
}

func meCmd(ctx *commandContext, args ...string) {
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		ctx.tab.Println("usage: /me [message...]")
		return
	}
	var dest string
	if ctx.chanState != nil {
		dest = ctx.chanState.channel
	} else if ctx.pmState != nil {
		dest = ctx.pmState.nick
	} else {
		ctx.tab.Println("ERROR: /me can only be used in channels and private messages")
		return
	}
	ctx.servConn.conn.Action(dest, msg)
	ctx.tab.Println(fmt.Sprintf(color("%s", LightGrey)+color(" *%s %s*", DarkGrey), now(), ctx.servState.user.nick, msg))
}

func joinCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		ctx.tab.Println("usage: /join [#channel]")
		return
	}
	ctx.servConn.conn.Join(args[0])
}

func partCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, strings.Join(args, " "), ctx.servState)
	} else {
		ctx.tab.Println("ERROR: /part only works for channels. Try /close")
	}
}

func noticeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /notice [#channel or nick] [message...]")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	ctx.tab.Println(fmt.Sprintf("%s *** %s: %s", now(), ctx.servState.user.nick, msg))
}

func privmsgCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 || args[0][0] == '#' {
		ctx.tab.Println("usage: /msg [nick] [message...]")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	pmState, ok := ctx.servState.privmsgs[nick]
	if !ok {
		pmState = &privmsgState{
			nick: nick,
		}
		pmState.tab = NewPrivmsgTab(ctx.servConn, ctx.servState, pmState)
	}

	ctx.servConn.conn.Privmsg(nick, msg)
	pmState.tab.Println(fmt.Sprintf("%s <%s> %s", now(), ctx.servState.user.nick, msg))
	mw.WindowBase.Synchronize(func() {
		checkErr(tabWidget.SetCurrentIndex(pmState.tab.Index()))
	})
}

func nickCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		ctx.tab.Println("usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func connectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case CONNECTION_EMPTY:
		ctx.tab.Println("ERROR: no network specified (use /server)")
	case CONNECTING, CONNECTION_START:
		ctx.tab.Println("ERROR: connection in progress: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))
	case CONNECTED:
		ctx.tab.Println("ERROR: already connected to: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))

	case DISCONNECTED, CONNECTION_ERROR:
		ctx.servConn.Connect(ctx.servState)
	}
}

func reconnectCmd(ctx *commandContext, args ...string) {
	ctx.tab.Println(`DOESN'T WORK RIGHT NOW BUT TYPING:
/disconnect
/connect
DOES WORK

(EVEN THOUGH THAT'S LITERALLY WHAT THIS FUNCTION DID)
`)
	return
	/*
		switch ctx.servState.connState {
		case CONNECTION_EMPTY:
			ctx.tab.Println("ERROR: no network specified (use /server)")
		case CONNECTING, CONNECTION_START:
			ctx.tab.Println("ERROR: connection in progress: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))

		case CONNECTED:
			disconnectCmd(ctx, args...)
			<-time.After(time.Second * 3)
			fallthrough
		case DISCONNECTED, CONNECTION_ERROR:
			ctx.servConn.retryConnectEnabled = true
			connectCmd(ctx, args...)
		}
	*/
}

func disconnectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case DISCONNECTED, CONNECTION_ERROR, CONNECTION_EMPTY:
		return
	case CONNECTING, CONNECTION_START, CONNECTED:
		ctx.servConn.retryConnectEnabled = false
		select {
		case <-ctx.servConn.cancelRetryConnect:
		default:
			close(ctx.servConn.cancelRetryConnect)
		}
		ctx.servConn.conn.Quit(strings.Join(args, " "))
	}
}

func quitCmd(ctx *commandContext, args ...string) {
	disconnectCmd(ctx, args...)
	for k, chanState := range ctx.servState.channels {
		chanState.tab.Close()
		delete(ctx.servState.channels, k)
	}
	for k, pmState := range ctx.servState.privmsgs {
		pmState.tab.Close()
		delete(ctx.servState.privmsgs, k)
	}
	if ctx.servState.channelList != nil {
		ctx.servState.channelList.Close()
		ctx.servState.channelList = nil
	}
	if len(tabs) > 1 {
		ctx.servState.tab.Close()
	}
}

func closeCmd(ctx *commandContext, args ...string) {
	if ctx.tab == ctx.servState.tab {
		quitCmd(ctx, args...)

		if len(tabs) == 1 {
			ctx.servState = &serverState{
				connState: CONNECTION_EMPTY,
				channels:  map[string]*channelState{},
				privmsgs:  map[string]*privmsgState{},
				user:      ctx.servState.user,
				tab:       ctx.servState.tab,
			}
			ctx.servState.tab.Update(ctx.servState)
		} else {
			ctx.servState.tab.Close()
		}
		return
	}
	if ctx.chanState != nil {
		partCmd(ctx, args...)
		delete(ctx.servState.channels, ctx.chanState.channel)
	} else if ctx.pmState != nil {
		delete(ctx.servState.privmsgs, ctx.pmState.nick)
	}
	ctx.tab.Close()
}

func modeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /mode [#channel or your nick] [mode] [nicks...]")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func clearCmd(ctx *commandContext, args ...string) {
	ctx.tab.Clear()
}

func topicCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /topic [new topic...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Topic(ctx.chanState.channel, args...)
	} else {
		ctx.tab.Println("ERROR: /topic can only be used in channels")
	}
}

func rejoinCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, "rejoining...", ctx.servState)
		ctx.servConn.conn.Join(ctx.chanState.channel)
	} else {
		ctx.tab.Println("ERROR: /rejoin only works for channels.")
	}
}

func kickCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /kick [nick] [(optional) reason...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Kick(ctx.chanState.channel, args[0], args[1:]...)
	} else {
		ctx.tab.Println("ERROR: /kick only works for channels.")
	}
}

func serverCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /server [host] [port (default 6667)]\r\n  ssl: /server [host] +[port (default 6697)]")
		return
	}
	hostname := args[0]
	port := 6667
	ssl := false
	if len(args) > 1 {
		portStr := args[1]
		if portStr[0] == '+' {
			ssl = true
			port = 6697
			portStr = portStr[1:]
		}
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}
	servState := &serverState{
		connState:   CONNECTION_EMPTY,
		hostname:    hostname,
		port:        port,
		ssl:         ssl,
		networkName: fmt.Sprintf("%s:%d", hostname, port),
		user: &userState{
			nick: ctx.servState.user.nick,
		},
		channels: map[string]*channelState{},
		privmsgs: map[string]*privmsgState{},
	}
	servConn := NewServerConnection(servState, func() {})
	if len(tabs) == 1 && ctx.servState != nil && ctx.servState.tab != nil && ctx.servState.connState == CONNECTION_EMPTY {
		servState.tab = ctx.servState.tab
	} else {
		servView := NewServerTab(servConn, servState)
		servState.tab = servView
	}
	servConn.Connect(servState)
}

func listCmd(ctx *commandContext, args ...string) {
	if ctx.servState.channelList == nil {
		ctx.servState.channelList = NewChannelList(ctx.servConn, ctx.servState)
	}
	if ctx.servState.channelList.complete {
		ctx.servState.channelList.Clear()
	}
	if !ctx.servState.channelList.inProgress {
		ctx.servConn.conn.Raw("LIST")
	}
}
