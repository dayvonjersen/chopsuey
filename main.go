package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fluffle/goirc/client"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	var (
		mw          *walk.MainWindow
		nickListBox *walk.ListBox
		textBuffer  *walk.TextEdit
		textInput   *walk.LineEdit
	)
	nickListBoxModel := &listboxModel{}

	printMessage := func(nick, msg string) {
		str := fmt.Sprintf("%s <%s> %s", time.Now().Format("3:04"), nick, msg)
		log.Println(str)
		textBuffer.AppendText(str + "\r\n")
	}

	host := "irc.lainchan.org"
	port := 6697
	ssl := true
	nick := "tso|testing"
	join := "#bots"

	irc := newConn(host, port, ssl, nick, join)
	irc.HandleFunc(client.PRIVMSG, func(c *client.Conn, l *client.Line) {
		printMessage(l.Nick, l.Args[1])
	})
	sendMessage := func(msg string) {
		irc.Privmsg(join, msg)
		printMessage(nick, msg)
	}

	// NAMES
	irc.HandleFunc("353", func(c *client.Conn, l *client.Line) {
		for _, nick := range strings.Split(l.Args[3], " ") {
			if nick != "" {
				nickListBoxModel.Items = append(nickListBoxModel.Items, nick)
				nickListBoxModel.PublishItemChanged(len(nickListBoxModel.Items) - 1)
			}
		}
	})

	MainWindow{
		AssignTo: &mw,
		Title:    "IRC",
		MinSize:  Size{480, 640},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			HSplitter{
				AlwaysConsumeSpace: true,
				Children: []Widget{
					TextEdit{
						MinSize:    Size{380, 640},
						AssignTo:   &textBuffer,
						ReadOnly:   true,
						Persistent: true,
					},
					ListBox{
						MinSize:    Size{100, 640},
						AssignTo:   &nickListBox,
						Model:      nickListBoxModel,
						Persistent: true,
					},
				},
			},
			LineEdit{
				AssignTo: &textInput,
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						sendMessage(textInput.Text())
						textInput.SetText("")
					}
				},
			},
		},
	}.Create()

	log.Println(irc.ConnectTo(host))
	irc.Raw("NAMES " + join)

	mw.Run()
}

type listboxModel struct {
	walk.ListModelBase
	Items []string
}

func (m *listboxModel) ItemCount() int {
	return len(m.Items)
}

func (m *listboxModel) Value(index int) interface{} {
	return m.Items[index]
}
