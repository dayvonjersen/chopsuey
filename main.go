package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var tabWidget *walk.TabWidget
var mw *walk.MainWindow

func main() {
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

	tabWidget.SetPersistent(true)

	cfg := getClientConfig()

	var tb *walk.TextEdit

	l := &tsoLogger{}
	l.LogFn = func(str string) {
		tb.AppendText(str + "\r\n")
	}

	logging.SetLogger(l)

	p, err := walk.NewTabPage()
	checkErr(err)
	p.SetTitle(cfg.Host)
	v := walk.NewVBoxLayout()
	p.SetLayout(v)
	b := NewBuilder(p)
	TextEdit{
		MinSize:    Size{480, 600},
		AssignTo:   &tb,
		ReadOnly:   true,
		Persistent: true,
	}.Create(b)
	tabWidget.Pages().Add(p)
	checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(p)))
	tabWidget.SaveState()

	servConn := newServerConnection(cfg)
	go func() {
		for {
			join := <-servConn.newChats
			mw.WindowBase.Synchronize(func() {
				newChatBoxTab(servConn, join)
			})
		}
	}()
	servConn.connect()

	mw.Run()
}

func newChatBoxTab(servConn *serverConnection, join string) {
	var (
		nickListBox *walk.ListBox
		textBuffer  *walk.TextEdit
		textInput   *walk.LineEdit
	)
	nickListBoxModel := &listboxModel{}

	chat, ok := servConn.chatBoxes[join]
	if !ok {
		log.Println("newChatBoxTab() called but user not on channel:", join)
		return
	}

	chat.printMessage = func(msg string) {
		textBuffer.AppendText(msg + "\r\n")
	}

	chat.sendMessage = func(msg string) {
		servConn.conn.Privmsg(join, msg)
		chat.printMessage(fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), servConn.cfg.Nick, msg))
	}

	chat.setNickList = func(nicks []string) {
		for _, nick := range nicks {
			nickListBoxModel.Items = append(nickListBoxModel.Items, nick)
			nickListBoxModel.PublishItemChanged(len(nickListBoxModel.Items) - 1)
		}
	}

	go func() {
		for {
			msg, ok := <-chat.messages
			if !ok {
				return
			}
			mw.WindowBase.Synchronize(func() {
				chat.printMessage(msg)
			})
		}
	}()
	servConn.chatBoxes[join] = chat

	page, err := walk.NewTabPage()
	checkErr(err)
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
				text := textInput.Text()
				if len(text) < 1 {
					return
				}
				if text[0] == '/' {
					parts := strings.Split(text[1:], " ")
					cmd := parts[0]
					var args []string
					if len(parts) > 1 {
						args = parts[1:]
					} else {
						args = []string{}
					}
					if cmdFn, ok := clientCommands[cmd]; ok {
						cmdFn(&clientContext{servConn, join}, args...)
					} else {
						log.Println("unrecognized command:", cmd)
					}
				} else {
					chat.sendMessage(text)
				}
				textInput.SetText("")
			}
		},
	}.Create(builder)

	checkErr(tabWidget.Pages().Add(page))
	checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(page)))
	tabWidget.SaveState()
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
