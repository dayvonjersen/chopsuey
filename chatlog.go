package main

import (
	"io"
	"os"
	"regexp"
	"time"
)

var invalidCharsInFilenamesRegex = regexp.MustCompile("[/<>:\"\\|?*]")

func NewChatLogger(filename string) func(string) {
	if !clientState.cfg.ChatLogsEnabled {
		return func(string) {}
	}

	filename = invalidCharsInFilenamesRegex.ReplaceAllString(filename, "_") + ".log"

	logger := func(message string) {
		f, err := os.OpenFile(CHATLOG_DIR+filename, os.O_CREATE|os.O_APPEND, os.ModePerm)
		checkErr(err)
		defer f.Close()
		io.WriteString(f, message+"\n")
	}

	// NOTE(tso): even if it's never called, the creation of a chat logger
	//            means that logging began, right?
	//            this might get annoying when joining channels by accident
	//            and having a log file written
	// -tso 7/11/2018 7:30:49 PM
	logger(time.Now().Format("-----------------------Mon Jan 2 15:04:05 -0700 MST 2006-----------------------"))

	return logger
}
