package main

import (
	"fmt"
	"strconv"
	"strings"
)

type clientContext struct {
	servConn *serverConnection
	channel  string
	cb       *chatBox
}

type clientCommand func(ctx *clientContext, args ...string)

var clientCommands map[string]clientCommand

func init() {
	clientCommands = map[string]clientCommand{
		"clear":  clearCmd,
		"close":  closeCmd,
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

		"raw": rawCmd,
	}
}

// for debug purposes only
func rawCmd(ctx *clientContext, args ...string) {
	ctx.servConn.conn.Raw(strings.Join(args, " "))
}

func meCmd(ctx *clientContext, args ...string) {
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		ctx.cb.printMessage("usage: /me [message...]")
		return
	}
	ctx.servConn.conn.Action(ctx.channel, msg)
	ctx.cb.printMessage(fmt.Sprintf("%s * %s %s", now(), ctx.servConn.Nick, msg))
}

func joinCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		ctx.cb.printMessage("usage: /join [#channel]")
		return
	}
	ctx.servConn.join(args[0])
}

func partCmd(ctx *clientContext, args ...string) {
	ctx.servConn.part(ctx.channel, strings.Join(args, " "))
}

func noticeCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 {
		ctx.cb.printMessage("usage: /notice [#channel or nick] [message...]")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	ctx.cb.printMessage(fmt.Sprintf("%s *** %s: %s", now(), ctx.servConn.Nick, msg))
}

func privmsgCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 || args[0][0] == '#' {
		ctx.cb.printMessage("usage: /msg [nick] [message...]")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	ctx.servConn.conn.Privmsg(nick, msg)

	cb := ctx.servConn.getChatBox(nick)
	if cb == nil {
		cb = ctx.servConn.createChatBox(nick, CHATBOX_PRIVMSG)
	}
	cb.printMessage(fmt.Sprintf("%s <%s> %s", now(), ctx.servConn.Nick, msg))
}

func nickCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 {
		ctx.cb.printMessage("usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func quitCmd(ctx *clientContext, args ...string) {
	ctx.servConn.retryConnectEnabled = false
	ctx.servConn.conn.Quit(strings.Join(args, " "))
	for _, cb := range ctx.servConn.chatBoxes {
		cb.close()
	}
}

func modeCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 {
		ctx.cb.printMessage("usage: /mode [#channel or your nick] [mode] [nicks...]")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func clearCmd(ctx *clientContext, args ...string) {
	ctx.cb.textBuffer.SetText("")
}

func topicCmd(ctx *clientContext, args ...string) {
	if len(args) < 1 {
		ctx.cb.printMessage("usage: /topic [new topic...]")
		return
	}
	ctx.servConn.conn.Topic(ctx.channel, args...)
}

func closeCmd(ctx *clientContext, args ...string) {
	if ctx.cb.boxType == CHATBOX_CHANNEL {
		partCmd(ctx, args...)
	} else {
		ctx.cb.close()
	}
}

func rejoinCmd(ctx *clientContext, args ...string) {
	if ctx.cb.boxType == CHATBOX_CHANNEL {
		ctx.servConn.join(ctx.cb.id)
	} else {
		ctx.cb.printMessage("ERROR: /rejoin only works for channels.")
	}
}

func kickCmd(ctx *clientContext, args ...string) {
	if len(args) < 1 {
		ctx.cb.printMessage("usage: /kick [nick] [(optional) reason...]")
		return
	}
	if ctx.cb.boxType == CHATBOX_CHANNEL {
		ctx.servConn.conn.Kick(ctx.cb.id, args[0], args[1:]...)
	} else {
		ctx.cb.printMessage("ERROR: /kick only works for channels.")
	}
}

func serverCmd(ctx *clientContext, args ...string) {
	if len(args) < 1 {
		ctx.cb.printMessage("usage: /server [host] [port (default 6667)]\r\n  ssl: /server [host] +[port (default 6697)]")
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
	cfg := &connectionConfig{
		Host:     host,
		Port:     port,
		Ssl:      ssl,
		Nick:     ctx.servConn.Nick,
		AutoJoin: []string{},
	}
	servConn := newServerConnection(cfg)
	servConn.connect()
}

func listCmd(ctx *clientContext, args ...string) {
	if ctx.servConn.channelList == nil {
		ctx.servConn.channelList = newChannelList(ctx.servConn)
	}
	if ctx.servConn.channelList.complete {
		ctx.servConn.channelList.Clear()
	}
	if !ctx.servConn.channelList.inProgress {
		ctx.servConn.conn.Raw("LIST")
	}
}
