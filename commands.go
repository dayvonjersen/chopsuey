package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type clientContext struct {
	servConn *serverConnection
	channel  string
}

type clientCommand func(ctx *clientContext, args ...string)

var clientCommands = map[string]clientCommand{
	"test": testCmd,
	"me":   meCmd,
	"join": joinCmd,
	"part": partCmd,
}

func testCmd(ctx *clientContext, args ...string) {
	log.Printf("%#v", ctx.servConn.chatBoxes[ctx.channel])
	log.Printf("%#v", ctx.servConn.chatBoxes[ctx.channel].nickList.StringSlice())
}

func meCmd(ctx *clientContext, args ...string) {
	msg := strings.Join(args, " ")
	chat := ctx.servConn.chatBoxes[ctx.channel]
	if len(args) == 0 {
		chat.messages <- "ERROR: missing message for /me"
		return
	}
	ctx.servConn.conn.Action(ctx.channel, msg)
	chat.messages <- fmt.Sprintf("%s * %s %s", time.Now().Format("15:04"), ctx.servConn.cfg.Nick, msg)
}

func joinCmd(ctx *clientContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		chat := ctx.servConn.chatBoxes[ctx.channel]
		chat.messages <- "usage: /join #channel"
		return
	}
	ctx.servConn.join(args[0])
}

func partCmd(ctx *clientContext, args ...string) {
	ctx.servConn.part(ctx.channel, strings.Join(args, " "))
}
