package main

import "log"

type clientContext struct {
	servConn *serverConnection
	channel  string
}

type clientCommand func(ctx *clientContext, args ...string)

var clientCommands = map[string]clientCommand{
	"test": testCmd,
}

func testCmd(ctx *clientContext, args ...string) {
	log.Println("hello world")
}
