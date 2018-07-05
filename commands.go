package main

import (
	"fmt"
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

		"raw":  rawCmd,
		"send": sendCmd,
	}
}

// for debug purposes only
func rawCmd(ctx *commandContext, args ...string) {
	ctx.servConn.conn.Raw(strings.Join(args, " "))
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
	ctx.tab.Println(fmt.Sprintf("%s * %s %s", now(), ctx.servState.user.nick, msg))
}

func joinCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		ctx.tab.Println("usage: /join [#channel]")
		return
	}
	ctx.servConn.Join(args[0], ctx.servState)
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
		ctx.servState.privmsgs[nick] = pmState
	}

	ctx.servConn.conn.Privmsg(nick, msg)
	pmState.tab.Println(fmt.Sprintf("%s <%s> %s", now(), ctx.servState.user.nick, msg))
	mw.WindowBase.Synchronize(func() {
		checkErr(tabWidget.SetCurrentIndex(pmState.tab.Id()))
	})
}

func nickCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		ctx.tab.Println("usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func quitCmd(ctx *commandContext, args ...string) {
	ctx.servConn.retryConnectEnabled = false
	ctx.servConn.conn.Quit(strings.Join(args, " "))
	for _, chanState := range ctx.servState.channels {
		chanState.tab.Close()
	}
	for _, pmState := range ctx.servState.privmsgs {
		pmState.tab.Close()
	}
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

func closeCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		partCmd(ctx, args...)
	} else {
		ctx.tab.Close()
	}
}

func rejoinCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Join(ctx.chanState.channel, ctx.servState)
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
	host := args[0]
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
		connected: false,
		hostname:  host,
		port:      port,
		ssl:       ssl,
		user: &userState{
			nick: ctx.servState.user.nick,
		},
		channels: map[string]*channelState{},
		privmsgs: map[string]*privmsgState{},
	}
	servConn := NewServerConnection(servState, func() {})
	servView := NewServerTab(servConn, servState)
	servState.tab = servView
	servConn.Connect()
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
