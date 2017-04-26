package main

import (
	"fmt"
	"strings"
	"time"
)

type clientContext struct {
	servConn *serverConnection
	channel  string
}

type clientCommand func(ctx *clientContext, args ...string)

var clientCommands = map[string]clientCommand{
	"test":   testCmd,
	"me":     meCmd,
	"join":   joinCmd,
	"part":   partCmd,
	"notice": noticeCmd,
	"msg":    privmsgCmd,
}

func testCmd(ctx *clientContext, args ...string) {
}

func meCmd(ctx *clientContext, args ...string) {
	msg := strings.Join(args, " ")
	chat := ctx.servConn.chatBoxes[ctx.channel]
	if len(args) == 0 {
		chat.messages <- "usage: /me [message...]"
		return
	}
	ctx.servConn.conn.Action(ctx.channel, msg)
	chat.messages <- fmt.Sprintf("%s * %s %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg)
}

func joinCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		chat := ctx.servConn.chatBoxes[ctx.channel]
		chat.messages <- "usage: /join [#channel]"
		return
	}
	ctx.servConn.join(args[0])
}

func partCmd(ctx *clientContext, args ...string) {
	ctx.servConn.part(ctx.channel, strings.Join(args, " "))
}

func noticeCmd(ctx *clientContext, args ...string) {
	chat := ctx.servConn.chatBoxes[ctx.channel]
	if len(args) < 2 {
		chat.messages <- "usage: /notice [#channel or nick] [message...]"
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	chat.messages <- fmt.Sprintf("%s *** %s: %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg)
}

func privmsgCmd(ctx *clientContext, args ...string) {
	if len(args) < 2 || args[0][0] == '#' {
		chat := ctx.servConn.chatBoxes[ctx.channel]
		chat.messages <- "usage: /msg [nick] [message...]"
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")
	chat, ok := ctx.servConn.chatBoxes[nick]
	if !ok {
		ctx.servConn.chatBoxes[nick] = newChatBox()
		ctx.servConn.newChats <- nick
		chat = ctx.servConn.chatBoxes[nick]
	}
	ctx.servConn.conn.Privmsg(nick, msg)
	go func() {
		chat.messages <- fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg)
	}()
}
