package main

import (
	"io"
	"log"
	"os"
)

const CHATLOG_DIR = "./chatlogs/"

func (cb *chatBox) logMessage(msg string) {
	if !clientCfg.ChatLogsEnabled {
		return
	}
	// hack
	if cb.servConn.networkName == cb.servConn.cfg.ServerString() {
		return
	}
	var fname string
	if cb.boxType == CHATBOX_SERVER {
		fname = cb.servConn.networkName + ".log"
	} else if cb.boxType == CHATBOX_CHANNEL || cb.boxType == CHATBOX_PRIVMSG {
		fname = cb.servConn.networkName + "-" + cb.id + ".log"
	} else {
		log.Println("attempt to log from unsupported chatBox: ", cb.id, "with boxType:", cb.boxType)
		log.Println("message not logged:", msg)
		return
	}
	f, err := os.OpenFile(CHATLOG_DIR+fname, os.O_CREATE|os.O_APPEND, os.ModePerm)
	checkErr(err)
	defer f.Close()
	io.WriteString(f, msg+"\n")
}
