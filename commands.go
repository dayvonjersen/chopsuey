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
}

func testCmd(ctx *clientContext, args ...string) {
	log.Println("hello world")
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
