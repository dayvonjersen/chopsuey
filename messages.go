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

func clientError(tab tabWithInput, msg ...string) {
	Println(CLIENT_ERROR, T(tab), msg)
}
func clientMessage(tab tabWithInput, msg ...string) {
	Println(CLIENT_MESSAGE, T(tab), msg)
}
func serverMessage(tab tabWithInput, msg ...string) {
	Println(SERVER_MESSAGE, T(tab), msg)
}
func serverError(tab tabWithInput, msg ...string) {
	Println(SERVER_ERROR, T(tab), msg)
}
func joinpartMessage(tab tabWithInput, msg ...string) {
	Println(JOINPART_MESSAGE, T(tab), msg)
}
func updateMessage(tab tabWithInput, msg ...string) {
	Println(UPDATE_MESSAGE, T(tab), msg)
}
func noticeMessage(tab tabWithInput, msg ...string) {
	Println(NOTICE_MESSAGE, T(tab), msg)
}
func actionMessage(tab tabWithInput, msg ...string) {
	Println(ACTION_MESSAGE, T(tab), msg)
}
func privateMessage(tab tabWithInput, msg ...string) {
	Println(PRIVATE_MESSAGE, T(tab), msg)
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
		time, nick, msg := now(), text[0], strings.Join(text[1:], " ")
		logmsg := time + "<" + nick + "> " + msg
		if highlight(nick, &msg) { // :point_left: :ok_hand: :joy: :100: :fire:
			t.Notify()
		}
		colorNick(&nick) // :point_left: :ok_hand: :joy: :100: :fire:
		for _, tab := range tabs {
			tab.Logln(logmsg)
			tab.Println(parseString(privateMsg(time, nick, msg)))
		}

	case ACTION_MESSAGE:
		time, nick := now(), text[0]
		msg := time + " *" + nick + msg + "*"
		logmsg := msg
		if highlight(nick, &msg) { // :point_left: :ok_hand: :joy: :100: :fire:
			t.Notify()
		}
		colorNick(&nick) // :point_left: :ok_hand: :joy: :100: :fire:
		for _, tab := range tabs {
			tab.Logln(msg)
			tab.Println(parseString(actionMsg(msg)))
		}

	default:
		log.Printf(`
		
		--------------------------------------------------------------------
		HEY!
		
		
		should this message be logged and displayed in the text buffer?


		%v
		

		??? default is yes...


		also msgType %d isn't defined. add it to messages.go`, text, msgType)

		text, styles := parseString(strings.Join(msg, " "))
		for _, tab := range tabs {
			tab.Logln(text)
			tab.Println(text, styles)
		}
	}
}

func clientMsg(text ...string) string {
	return color(now(), LightGray) + strings.Join(text, " ")
}

func clientErrorMsg(text ...string) string {
	return color(now()+" "+strings.Join(text, " "), Red)
}

func serverMsg(text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for server message: want 2 got %d:\n%v",
			len(text), text)
	}
	return color(now()+" "+color("ERROR("+text[0]+")", White, Red)+": "+strings.Join(text[1:], " "), Red)
}

func serverErrorMsg(text ...string) string {
	if len(text) < 2 {
		return fmt.Sprintf("wrong argument count for server error: want 2 got %d:\n%v",
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

func actionMsg(text ...string) string {
	return color(now(), LightGray) +
		" " + color("*"+strings.Join(text, " ")+"*", Blue)
}

func privateMsg(text ...string) string {
	nick := text[0]
	return color(now(), LightGray) +
		" " + color(nick, DarkGrey) + " " + strings.Join(text[1:], " ")
}
