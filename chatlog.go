package main

import (
	"io"
	"os"
	"regexp"
)

const CHATLOG_DIR = "./chatlogs/"

var re = regexp.MustCompile("[/<>:\"\\|?*]")

func NewChatLogger(filename string) func(string) {
	if !clientCfg.ChatLogsEnabled {
		return func(string) {}
	}

	filename = re.ReplaceAllString(filename, "_") + ".log"

	return func(message string) {
		f, err := os.OpenFile(CHATLOG_DIR+filename, os.O_CREATE|os.O_APPEND, os.ModePerm)
		checkErr(err)
		defer f.Close()
		io.WriteString(f, message+"\n")
	}

	/*
		NOTE(tso): bad idea not worth it tbh
		daily := <-time.After(0)

		go func() {
			for {
				<-daily
				logger(t.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))

				t := time.Now()
				year, month, day := t.Date()
				daily = time.After(time.Date(year, month, day, 23, 59, 59, 1000, t.Location()))
			}
		}()

		return logger
	*/
}
