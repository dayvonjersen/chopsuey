package main

import (
	"fmt"
	"log"
	"time"

	goirc "github.com/fluffle/goirc/client"
	"github.com/kr/pretty"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func printf(args ...interface{}) {
	s := ""
	for _, x := range args {
		s += fmt.Sprintf("%# v", pretty.Formatter(x))
	}
	log.Print(s)
}

func debugPrint(l *goirc.Line) {
	printf(&goirc.Line{Nick: l.Nick, Ident: l.Ident, Host: l.Host, Src: l.Src, Cmd: l.Cmd, Raw: l.Raw, Args: l.Args})
}

func now() string {
	return time.Now().Format(clientCfg.TimeFormat)
}

func isChannel(channel string) bool {
	// NOTE(tso): I've never seen a local channel (&) before but I'd like to join one someday.
	return channel[0] == '#' || channel[0] == '&'
}

func serverAddr(hostname string, port int) string {
	return fmt.Sprintf("%s:%d", hostname, port)
}
