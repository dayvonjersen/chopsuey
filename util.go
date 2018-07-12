package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kr/pretty"
)

func printf(args ...interface{}) {
	s := ""
	for _, x := range args {
		s += fmt.Sprintf("%# v", pretty.Formatter(x))
	}
	log.Print(s)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func now() string {
	return time.Now().Format(clientCfg.TimeFormat)
}
