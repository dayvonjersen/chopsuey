package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fluffle/goirc/client"
	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	host := "chopstick"
	port := 6667
	ssl := false
	nick := "tso|testing"
	join := "#test"
	var (
		mw          *walk.MainWindow
		tabWidget   *walk.TabWidget
		nickListBox *walk.ListBox
		textBuffer  *walk.TextEdit
		textInput   *walk.LineEdit
	)
	nickListBoxModel := &listboxModel{}

	MainWindow{
		AssignTo: &mw,
		Title:    "IRC",
		MinSize:  Size{480, 640},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			TabWidget{
				AssignTo: &tabWidget,
			},
		},
	}.Create()

	var tb *walk.TextEdit

	l := &tsoLogger{}
	l.LogFn = func(str string) {
		tb.AppendText(str + "\r\n")
	}

	logging.SetLogger(l)

	var p *walk.TabPage
	p, _ = walk.NewTabPage()
	p.SetTitle(host)
	v := walk.NewVBoxLayout()
	p.SetLayout(v)
	b := NewBuilder(p)
	TextEdit{
		MinSize:    Size{480, 640},
		AssignTo:   &tb,
		ReadOnly:   true,
		Persistent: true,
	}.Create(b)
	tabWidget.Pages().Add(p)

	printMessage := func(nick, msg string) {
		str := fmt.Sprintf("%s <%s> %s", time.Now().Format("3:04"), nick, msg)
		textBuffer.AppendText(str + "\r\n")
	}

	irc := newConn(host, port, ssl, nick, join)
	irc.HandleFunc(client.PRIVMSG, func(c *client.Conn, l *client.Line) {
		printMessage(l.Nick, l.Args[1])
	})
	sendMessage := func(msg string) {
		irc.Privmsg(join, msg)
		printMessage(nick, msg)
	}

	var page *walk.TabPage
	page, _ = walk.NewTabPage()
	page.SetTitle(join)
	vbox := walk.NewVBoxLayout()
	page.SetLayout(vbox)
	builder := NewBuilder(page)
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
	}.Create(builder)
	LineEdit{
		AssignTo: &textInput,
		OnKeyDown: func(key walk.Key) {
			if key == walk.KeyReturn {
				sendMessage(textInput.Text())
				textInput.SetText("")
			}
		},
	}.Create(builder)

	tabWidget.Pages().Add(page)

	// NAMES
	irc.HandleFunc("353", func(c *client.Conn, l *client.Line) {
		for _, nick := range strings.Split(l.Args[3], " ") {
			if nick != "" {
				nickListBoxModel.Items = append(nickListBoxModel.Items, nick)
				nickListBoxModel.PublishItemChanged(len(nickListBoxModel.Items) - 1)
			}
		}
	})

	checkErr(irc.ConnectTo(host))

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

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
