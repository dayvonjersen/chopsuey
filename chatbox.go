package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	CHATBOX_SERVER = iota
	CHATBOX_CHANNEL
	CHATBOX_PRIVMSG
	CHATBOX_CHANLIST
)

type chatBox struct {
	boxType          int
	id               string
	nickList         *nickList
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	servConn         *serverConnection
	textBuffer       *walk.TextEdit
	textInput        *walk.LineEdit
	topicInput       *walk.LineEdit
	title            string
	tabPage          *walk.TabPage
	msgHistory       []string
	msgHistoryIdx    int
}

func (cb *chatBox) printMessage(msg string) {
	mw.WindowBase.Synchronize(func() {
		cb.textBuffer.AppendText(msg + "\r\n")
	})
}

func (cb *chatBox) sendMessage(msg string) {
	cb.servConn.conn.Privmsg(cb.id, msg)
	cb.printMessage(fmt.Sprintf("%s <%s> %s", time.Now().Format("15:04"), cb.servConn.cfg.Nick, msg))
}

func (cb *chatBox) updateNickList() {
	mw.WindowBase.Synchronize(func() {
		cb.nickListBoxModel.Items = cb.nickList.StringSlice()
		cb.nickListBoxModel.PublishItemsReset()
	})
}

func (cb *chatBox) close() {
	checkErr(tabWidget.Pages().Remove(cb.tabPage))
	tabWidget.SaveState()
}

func newChatBox(servConn *serverConnection, id string, boxType int) *chatBox {
	cb := &chatBox{
		boxType:          boxType,
		id:               id,
		nickList:         &nickList{Mu: &sync.Mutex{}},
		nickListBox:      &walk.ListBox{},
		nickListBoxModel: &listBoxModel{},
		servConn:         servConn,
		textBuffer:       &walk.TextEdit{},
		textInput:        &walk.LineEdit{},
		topicInput:       &walk.LineEdit{},
		title:            id,
		msgHistory:       []string{},
		msgHistoryIdx:    0,
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		cb.tabPage, err = walk.NewTabPage()
		checkErr(err)
		cb.tabPage.SetTitle(cb.title)
		cb.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(cb.tabPage)

		LineEdit{
			AssignTo: &cb.topicInput,
			ReadOnly: true,
		}.Create(builder)
		HSplitter{
			Children: []Widget{
				TextEdit{
					MaxSize:            Size{340, 460},
					MinSize:            Size{340, 460},
					AssignTo:           &cb.textBuffer,
					ReadOnly:           true,
					AlwaysConsumeSpace: true,
					Persistent:         true,
				},
				ListBox{
					MaxSize:            Size{100, 460},
					MinSize:            Size{100, 460},
					AssignTo:           &cb.nickListBox,
					Model:              cb.nickListBoxModel,
					AlwaysConsumeSpace: true,
					Persistent:         true,
				},
			},
		}.Create(builder)
		LineEdit{
			AssignTo: &cb.textInput,
			OnKeyDown: func(key walk.Key) {
				if key == walk.KeyReturn {
					text := cb.textInput.Text()
					if len(text) < 1 {
						return
					}
					cb.msgHistory = append(cb.msgHistory, text)
					cb.msgHistoryIdx = len(cb.msgHistory) - 1
					if text[0] == '/' {
						parts := strings.Split(text[1:], " ")
						cmd := parts[0]
						if cmd[0] == '/' {
							cb.sendMessage(cmd)
						} else {
							var args []string
							if len(parts) > 1 {
								args = parts[1:]
							} else {
								args = []string{}
							}
							if cmdFn, ok := clientCommands[cmd]; ok {
								cmdFn(&clientContext{servConn: cb.servConn, channel: cb.id, cb: cb}, args...)
							} else {
								cb.printMessage("unrecognized command: " + cmd)
							}
						}
					} else {
						cb.sendMessage(text)
					}
					cb.textInput.SetText("")
				} else if key == walk.KeyUp {
					if len(cb.msgHistory) > 0 {
						text := cb.msgHistory[cb.msgHistoryIdx]
						cb.textInput.SetText(text)
						cb.textInput.SetTextSelection(len(text), len(text))
						cb.msgHistoryIdx--
						if cb.msgHistoryIdx < 0 {
							cb.msgHistoryIdx = 0
						}
					}
				} else if key == walk.KeyDown {
					if len(cb.msgHistory) > 0 {
						cb.msgHistoryIdx++
						if cb.msgHistoryIdx <= len(cb.msgHistory)-1 {
							text := cb.msgHistory[cb.msgHistoryIdx]
							cb.textInput.SetText(text)
							cb.textInput.SetTextSelection(len(text), len(text))
						} else {
							cb.textInput.SetText("")
							cb.msgHistoryIdx = len(cb.msgHistory) - 1
						}
					}
				}
			},
			OnKeyUp: func(key walk.Key) {
				if key == walk.KeyUp || key == walk.KeyDown {
					text := cb.textInput.Text()
					cb.textInput.SetTextSelection(len(text), len(text))
				}
			},
			OnKeyPress: func(key walk.Key) {
				if key == walk.KeyUp || key == walk.KeyDown {
					text := cb.textInput.Text()
					cb.textInput.SetTextSelection(len(text), len(text))
				}
			},
		}.Create(builder)

		checkErr(tabWidget.Pages().Add(cb.tabPage))
		checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(cb.tabPage)))
		tabWidget.SaveState()
	})

	return cb
}

type listBoxModel struct {
	walk.ListModelBase
	Items []string
}

func (m *listBoxModel) ItemCount() int {
	return len(m.Items)
}

func (m *listBoxModel) Value(index int) interface{} {
	return m.Items[index]
}
