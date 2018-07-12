// here be dragons
package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

const (
	CLIENT_MESSAGE = iota
	CLIENT_ERROR
	SERVER_MESSAGE
	SERVER_ERROR
	JOINPART_MESSAGE
	UPDATE_MESSAGE
	NOTICE_MESSAGE
	ACTION_MESSAGE
	PRIVATE_MESSAGE
)

func clientError(tab tabWithInput, msg ...string) {
	Println(CLIENT_ERROR, T(tab), msg...)
}
func clientMessage(tab tabWithInput, msg ...string) {
	Println(CLIENT_MESSAGE, T(tab), msg...)
}
func serverMessage(tab tabWithInput, msg ...string) {
	Println(SERVER_MESSAGE, T(tab), msg...)
}
func serverError(tab tabWithInput, msg ...string) {
	Println(SERVER_ERROR, T(tab), msg...)
}
func joinpartMessage(tab tabWithInput, msg ...string) {
	Println(JOINPART_MESSAGE, T(tab), msg...)
}
func updateMessage(tab tabWithInput, msg ...string) {
	Println(UPDATE_MESSAGE, T(tab), msg...)
}
func noticeMessage(tab tabWithInput, msg ...string) {
	Println(NOTICE_MESSAGE, T(tab), msg...)
}
func actionMessage(tab tabWithInput, msg ...string) {
	Println(ACTION_MESSAGE, T(tab), msg...)
}
func privateMessage(tab tabWithInput, msg ...string) {
	Println(PRIVATE_MESSAGE, T(tab), msg...)
}

func T(tab ...tabWithInput) (tabs []tabWithInput) { return } // expected type, found ILLEGAL

func Println(msgType int, tabs []tabWithInput, msg ...string) {

	if len(msg) == 0 {
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
			case UPDATE_MESSAGE:
				return "UPDATE_MESSAGE"
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
		for _, tab := range tabs {
			tab.Println(parseString(clientMsg(msg...)))
		}

	case CLIENT_ERROR:
		for _, tab := range tabs {
			tab.Errorln(parseString(clientErrorMsg(msg...)))
		}

	case SERVER_MESSAGE:
		text, styles := parseString(serverMsg(msg...))
		for _, tab := range tabs {
			tab.Logln(text)
			tab.Println(text, styles)
		}

	case SERVER_ERROR:
		text, styles := parseString(serverErrorMsg(msg...))
		for _, tab := range tabs {
			tab.Logln(text)
			tab.Errorln(text, styles)
		}

	case JOINPART_MESSAGE:
		if !clientCfg.HideJoinParts {
			text, styles := parseString(joinpartMsg(msg...))
			for _, tab := range tabs {
				tab.Logln(text)
				tab.Println(text, styles)
			}
		}

	case UPDATE_MESSAGE:
		// TODO(tso): option to hide?
		text, styles := parseString(joinpartMsg(msg...))
		for _, tab := range tabs {
			tab.Logln(text)
			tab.Println(text, styles)
		}

	case NOTICE_MESSAGE:

		for _, tab := range tabs {
			tab.Notify()
			tab.Logln("*** NOTICE: " + strings.Join(msg, " "))
			tab.Println(parseString(noticeMsg(msg...)))
		}

	case PRIVATE_MESSAGE:
		time, nick, msg := now(), msg[0], strings.Join(msg[1:], " ")
		logmsg := time + "<" + nick + "> " + msg
		hl := highlight(nick, &msg)
		colorNick(&nick)
		for _, tab := range tabs {
			if hl {
				tab.Notify()
			}
			tab.Logln(logmsg)
			tab.Println(parseString(privateMsg(hl, time, nick, msg)))
		}

	case ACTION_MESSAGE:
		logmsg := now() + " *" + strings.Join(msg, " ") + "*"
		time, nick, msg := now(), msg[0], strings.Join(msg[1:], " ")
		hl := highlight(nick, &msg)
		colorNick(&nick)
		for _, tab := range tabs {
			if hl {
				tab.Notify()
			}
			tab.Logln(logmsg)
			tab.Println(parseString(actionMsg(hl, time, nick, msg)))
		}

	default:
		log.Printf(`
		
		--------------------------------------------------------------------
		HEY!
		
		
		should this message be logged and displayed in the text buffer?


		%v
		

		??? default is yes...


		also msgType %d isn't defined. add it to messages.go`, msg, msgType)

		text, styles := parseString(strings.Join(msg, " "))
		for _, tab := range tabs {
			tab.Logln(text)
			tab.Println(text, styles)
		}
	}
}

func clientMsg(text ...string) string {
	return color(now(), LightGray) + " " + strings.Join(text, " ")
}

func clientErrorMsg(text ...string) string {
	return color(now()+" "+strings.Join(text, " "), Red)
}

func serverErrorMsg(text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for server error: want 2 got %d:\n%v",
			len(text), text)
	}
	return color(now()+" "+color("ERROR("+text[0]+")", White, Red)+": "+strings.Join(text[1:], " "), Red)
}

func serverMsg(text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for server message: want 2 got %d:\n%v",
			len(text), text)
	}
	return color(now()+" "+text[0]+": "+strings.Join(text[1:], " "), LightGray)
}

func joinpartMsg(text ...string) string {
	return color(now(), LightGray) + italic(color(strings.Join(text, " "), Orange))
}

func updateMsg(text ...string) string {
	return color(now(), LightGray) + italic(color(strings.Join(append([]string{"..."}, text...), " "), Orange))
}

func noticeMsg(text ...string) string {
	if len(text) < 3 {
		return fmt.Sprintf("wrong argument count for notice: want 3, got %d:\n%v", len(text), text)
	}
	return color(now(), LightGray) +
		color("NOTICE", White, Orange) + "(" + text[0] + "->" + text[1] + "): " +
		strings.Join(text, " ")
}

func actionMsg(hl bool, text ...string) string {
	line := color(now(), LightGray) + " "
	if hl {
		line += bold(color(" ..! ", Orange, Yellow))
	}
	return line + " " + "*" + strings.Join(text, " ") + "*"
}

func privateMsg(hl bool, text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for notice: want 2, got %d:\n%v", len(text), text)
	}
	nick := text[0]
	line := color(now(), LightGray) + " "
	if hl {
		line += bold(color(" ..! ", Orange, Yellow))
	}
	return line + nick + " " + strings.Join(text[1:], " ")

}

func highlight(nick string, msg *string) bool {
	// NOTE(tso): we can modify msg in-place that's why the function signature
	//            is like that but for now a marker next to the line is enough.
	//            Visual choice, not a programatic one.
	// -tso 7/12/2018 9:44:44 AM
	// NOTE(tso): not using compiled regexp here because user's nick can change
	//            unless recompiling a new one when the nick changes will really
	//            give that much of a performance increase
	// -tso 7/10/2018 6:58:36 AM
	m, _ := regexp.MatchString(`\b@*`+regexp.QuoteMeta(nick)+`(\b|[^\w])`, *msg)
	return m
}

func colorNick(nick *string) {
	// TODO(tso): different colors for nicks
	*nick = color(*nick, DarkGrey)
}
