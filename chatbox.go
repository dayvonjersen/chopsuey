package main

import (
	"fmt"
	"strings"

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
	textInput        *MyLineEdit
	topicInput       *walk.LineEdit
	title            string
	tabPage          *walk.TabPage
	msgHistory       []string
	msgHistoryIdx    int
	tabComplete      *tabComplete
}

func (cb *chatBox) printMessage(msg string) {
	mw.WindowBase.Synchronize(func() {
		cb.textBuffer.AppendText(msg + "\r\n")
		cb.logMessage(msg)
		if cb.tabPage != getCurrentTab() {
			if !strings.HasPrefix(cb.tabPage.Title(), "* ") {
				cb.tabPage.SetTitle("* " + cb.tabPage.Title())
			}
		}
	})
}

func (cb *chatBox) sendMessage(msg string) {
	if !(cb.boxType == CHATBOX_CHANNEL || cb.boxType == CHATBOX_PRIVMSG) {
		return
	}
	cb.servConn.conn.Privmsg(cb.id, msg)
	nick := newNick(cb.servConn.Nick)
	if cb.boxType == CHATBOX_CHANNEL {
		nick = cb.nickList.Get(cb.servConn.Nick)
	}
	cb.printMessage(fmt.Sprintf("%s <%s> %s", now(), nick, msg))
}

func (cb *chatBox) updateNickList() {
	mw.WindowBase.Synchronize(func() {
		cb.nickListBoxModel.Items = cb.nickList.StringSlice()
		cb.nickListBoxModel.PublishItemsReset()
	})
}

func (cb *chatBox) close() {
	mw.WindowBase.SetSuspended(true)
	defer mw.WindowBase.SetSuspended(false)
	checkErr(tabWidget.Pages().Remove(cb.tabPage))
	checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Len() - 1))
	tabWidget.SaveState()
}

type tabComplete struct {
	Active  bool
	Entries []string
	Index   int
}

func newChatBox(servConn *serverConnection, id string, boxType int) *chatBox {
	cb := &chatBox{
		boxType:       boxType,
		id:            id,
		servConn:      servConn,
		textBuffer:    &walk.TextEdit{},
		title:         id,
		msgHistory:    []string{},
		msgHistoryIdx: 0,
		nickList:      newNickList(),
		tabComplete:   &tabComplete{},
	}
	if cb.boxType == CHATBOX_SERVER {
	}
	if cb.boxType == CHATBOX_CHANNEL {
		cb.nickListBox = &walk.ListBox{}
		cb.nickListBoxModel = &listBoxModel{}
		cb.topicInput = &walk.LineEdit{}
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		cb.tabPage, err = walk.NewTabPage()
		checkErr(err)
		cb.tabPage.SetTitle(cb.title)
		cb.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(cb.tabPage)

		if cb.boxType == CHATBOX_CHANNEL {
			LineEdit{
				AssignTo: &cb.topicInput,
				ReadOnly: true,
			}.Create(builder)
			var hsplit *walk.Splitter
			HSplitter{
				AssignTo: &hsplit,
				Children: []Widget{
					TextEdit{
						AssignTo:           &cb.textBuffer,
						ReadOnly:           true,
						AlwaysConsumeSpace: true,
						VScroll:            true,
						MaxLength:          0x7FFFFFFE,
						StretchFactor:      3,
					},
					ListBox{
						StretchFactor:      1,
						AssignTo:           &cb.nickListBox,
						Model:              cb.nickListBoxModel,
						AlwaysConsumeSpace: false,
						OnItemActivated: func() {
							nick := newNick(cb.nickListBoxModel.Items[cb.nickListBox.CurrentIndex()])
							box := cb.servConn.getChatBox(nick.name)
							if box == nil {
								cb.servConn.createChatBox(nick.name, CHATBOX_PRIVMSG)
							} else {
								checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(box.tabPage)))
							}
						},
					},
				},
				AlwaysConsumeSpace: true,
			}.Create(builder)
			checkErr(hsplit.SetHandleWidth(1))
		} else if cb.boxType == CHATBOX_SERVER || cb.boxType == CHATBOX_PRIVMSG {
			TextEdit{
				AssignTo:           &cb.textBuffer,
				ReadOnly:           true,
				AlwaysConsumeSpace: true,
				Persistent:         true,
				VScroll:            true,
				MaxLength:          0x7FFFFFFE,
			}.Create(builder)
		}

		cb.textInput = newMyLineEdit(cb.tabPage)
		cb.textInput.KeyDown().Attach(func(key walk.Key) {
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
		})

		cb.textInput.KeyUp().Attach(func(key walk.Key) {
			if key == walk.KeyUp || key == walk.KeyDown {
				text := cb.textInput.Text()
				cb.textInput.SetTextSelection(len(text), len(text))
			}
		})

		cb.textInput.KeyPress().Attach(func(key walk.Key) {
			if key == walk.KeyUp || key == walk.KeyDown {
				text := cb.textInput.Text()
				cb.textInput.SetTextSelection(len(text), len(text))
			} else if key == walk.KeyTab && cb.boxType == CHATBOX_CHANNEL {
				text := strings.Split(cb.textInput.Text(), " ")
				if cb.tabComplete.Active {
					cb.tabComplete.Index++
					if cb.tabComplete.Index >= len(cb.tabComplete.Entries) {
						cb.tabComplete.Index = 0
					}
				} else {
					term := text[len(text)-1]
					res := cb.nickList.Search(term)
					res = append(res, term)
					cb.tabComplete = &tabComplete{
						Active:  true,
						Entries: res,
						Index:   0,
					}
				}
				text = append(text[:len(text)-1], cb.tabComplete.Entries[cb.tabComplete.Index])
				t := strings.Join(text, " ")
				cb.textInput.SetText(t)
				cb.textInput.SetTextSelection(len(t), len(t))
			} else {
				if cb.tabComplete.Active {
					cb.tabComplete = &tabComplete{}
				}
			}
		})

		checkErr(cb.tabPage.Children().Add(cb.textInput))
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
