// here be dragons
package main

import (
	"fmt"
	"log"
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

func msgTypeString(t int) string {
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
}

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

type highlighterFn func(nick, msg string) bool

func noticeMessageWithHighlight(tab tabWithInput, hl highlighterFn, msg ...string) {
	PrintlnWithHighlight(NOTICE_MESSAGE, hl, T(tab), msg...)
}
func actionMessageWithHighlight(tab tabWithInput, hl highlighterFn, msg ...string) {
	PrintlnWithHighlight(ACTION_MESSAGE, hl, T(tab), msg...)
}
func privateMessageWithHighlight(tab tabWithInput, hl highlighterFn, msg ...string) {
	PrintlnWithHighlight(PRIVATE_MESSAGE, hl, T(tab), msg...)
}

func T(tabs ...tabWithInput) []tabWithInput { return tabs } // expected type, found ILLEGAL

func PrintlnWithHighlight(msgType int, hl highlighterFn, tabs []tabWithInput, msg ...string) {
	switch msgType {
	case NOTICE_MESSAGE:
		for _, tab := range tabs {
			tab.Logln(now() + " *** NOTICE: " + strings.Join(msg, " "))

			tab.Notify()

			h := false
			if len(msg) >= 3 {
				h = hl(msg[1], strings.Join(msg[2:], " "))
			}
			tab.Println(parseString(noticeMsg(h, msg...)))
		}

	case PRIVATE_MESSAGE:
		nick, msg := msg[0], strings.Join(msg[1:], " ")
		logmsg := now() + " <" + nick + "> " + msg
		h := hl(nick, msg)
		colorNick(&nick)
		for _, tab := range tabs {
			if h {
				tab.Notify()
			}
			tab.Logln(logmsg)
			tab.Println(parseString(privateMsg(h, nick, msg)))
		}

	case ACTION_MESSAGE:
		logmsg := now() + " *" + strings.Join(msg, " ") + "*"
		nick, msg := msg[0], strings.Join(msg[1:], " ")
		h := hl(nick, msg)
		colorNick(&nick)
		for _, tab := range tabs {
			if h {
				tab.Notify()
			}
			tab.Logln(logmsg)
			tab.Println(parseString(actionMsg(h, nick, msg)))
		}
	default:
		log.Println("highlighting unsupported for msgType %v", msgTypeString(msgType))
	}
}

func Println(msgType int, tabs []tabWithInput, msg ...string) {
	if len(msg) == 0 {
		log.Printf("tried to print an empty line of type %v", msgTypeString(msgType))
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
		if !clientState.cfg.HideJoinParts {
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
			tab.Logln(now() + " *** NOTICE: " + strings.Join(msg, " "))
			tab.Println(parseString(noticeMsg(false, msg...)))
		}

	case PRIVATE_MESSAGE:
		nick, msg := msg[0], strings.Join(msg[1:], " ")
		logmsg := now() + " <" + nick + "> " + msg
		colorNick(&nick)
		for _, tab := range tabs {
			tab.Logln(logmsg)
			tab.Println(parseString(privateMsg(false, nick, msg)))
		}

	case ACTION_MESSAGE:
		logmsg := now() + " *" + strings.Join(msg, " ") + "*"
		nick, msg := msg[0], strings.Join(msg[1:], " ")
		colorNick(&nick)
		for _, tab := range tabs {
			tab.Logln(logmsg)
			tab.Println(parseString(actionMsg(false, nick, msg)))
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
		return fmt.Sprintf("wrong argument count for server error: want 2 got %d:\n%#v",
			len(text), text)
	}
	return color(now(), Red) + " " +
		color("ERROR("+text[0]+")", White, Red) + " " +
		color(strings.Join(text[1:], " "), Red)
}

func serverMsg(text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for server message: want 2 got %d:\n%#v",
			len(text), text)
	}
	return color(now()+" "+text[0]+": "+strings.Join(text[1:], " "), DarkGray)
}

func joinpartMsg(text ...string) string {
	return color(now(), LightGray) + " " + italic(color(strings.Join(text, " "), Orange))
}

func updateMsg(text ...string) string {
	return color(now(), LightGray) + " " + color(strings.Join(append([]string{"..."}, text...), " "), LightGrey)
}

func noticeMsg(hl bool, text ...string) string {
	if len(text) < 3 {
		return fmt.Sprintf("wrong argument count for notice: want 3, got %d:\n%v", len(text), text)
	}
	line := color(now(), LightGray) +
		color("NOTICE", White, Orange) +
		color("("+text[0]+"->"+text[1]+"): ", Orange)
	if hl {
		line += bold(color(" ..! ", Orange, Yellow))
	}
	return line + strings.Join(text[1:], " ")
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

func colorNick(nick *string) {
	// TODO(tso): different colors for nicks
	// TODO(tso): npm install left-pad
	*nick = color(*nick, DarkGrey)
}
