package main

import (
	"log"
	"strings"
)

const (
	CLIENT_MESSAGE = iota
	CLIENT_ERROR
	SERVER_MESSAGE
	SERVER_ERROR
	JOINPART_MESSAGE
	NOTICE_MESSAGE
	ACTION_MESSAGE
	PRIVATE_MESSAGE
)

func Println(tab tabWithInput, msgType int, text ...string) {
	if len(text) == 0 {
		log.Printf("tried to print an empty line of type %v", func(t int) string {
			switch t {
			case CLIENT_MESSAGE:
				return "CLIENT_MESSAGE"
			case CLIENT_ERROR:
				return "CLIENT_ERROR"
			case SERVER_MESSAGE:
				return "SERVER_MESSAGE"
			case SERVER_ERROR:
				return "SERVER_ERROR"
			case JOINPART_MESSAGE:
				return "JOINPART_MESSAGE"
			case NOTICE_MESSAGE:
				return "NOTICE_MESSAGE"
			case ACTION_MESSAGE:
				return "ACTION_MESSAGE"
			case PRIVATE_MESSAGE:
				return "PRIVATE_MESSAGE"
			}
			return "(unknown)"
		}(msgType))
		return
	}

	switch msgType {
	case CLIENT_MESSAGE:
		tab.Println(clientMsg(text...))

	case CLIENT_ERROR:
		tab.Errorln(clientErrorMsg(text...))

	case SERVER_ERROR:
		tab.Logln(strings.Join(text, " "))
		tab.Errorln(serverErrorMsg(text...))

	case SERVER_MESSAGE:
		tab.Logln(strings.Join(text, " "))
		tab.Println(serverMsg(text...))

	case JOINPART_MESSAGE:
		if !clientCfg.HideJoinParts {
			tab.Logln(strings.Join(text, " "))
			tab.Println(joinpartMsg(text...))
		}

	case NOTICE_MESSAGE:
		tab.Logln("*** NOTICE: " + strings.Join(text, " "))
		tab.Println(noticeMsg(text...))

	case PRIVATE_MESSAGE:
		tab.Logln("<" + text[0] + "> " + strings.Join(text[1:], " "))
		tab.Println(privateMsg(text...))

	case ACTION_MESSAGE:
		tab.Logln("*" + strings.Join(text, " ") + "*")
		tab.Println(actionMsg(text...))

	default:
		log.Printf(`
		
		--------------------------------------------------------------------
		HEY!
		
		
		should this message be logged and displayed in the text buffer?


		%v
		

		??? default is yes...


		also msgType %d isn't defined. add it to messages.go`, text, msgType)

		tab.Logln(strings.Join(text, " "))
		tab.Println(strings.Join(text, " "))
	}
}

func clientMsg(text ...string) string {
	return color(now(), LightGray) + strings.Join(text, " ")
}

func clientErrorMsg(text ...string) string {
	return color(now()+" "+strings.Join(text, " "), Red)
}

func serverMsg(text ...string) string {
	return color(now()+" "+strings.Join(text, " "), LightGray)
}

func serverErrorMsg(text ...string) string {
	return color(now()+" "+strings.Join(text, " "), Red)
}

func joinpartMsg(text ...string) string {
	return color(now(), LightGray) + italic(color(strings.Join(text, " "), Orange))
}

func noticeMsg(text ...string) string {
	return color(now(), LightGray) +
		" " + color("***", White, Orange) +
		" " + strings.Join(text, " ")
}

func actionMsg(text ...string) string {
	return color(now(), LightGray) +
		" " + color("*"+strings.Join(text, " ")+"*", Blue)
}

func privateMsg(text ...string) string {
	nick := text[0]
	return color(now(), LightGray) +
		" " + color(nick, DarkGrey) + " " + strings.Join(text[1:], " ")
}
