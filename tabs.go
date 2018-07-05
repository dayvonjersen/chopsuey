package main

import (
	"fmt"
	"strings"

	"github.com/lxn/walk"
)

type tabView interface {
	Id() int
	Title() string
	StatusText() string
	Focus()
	Close()
}

type tabViewCommon struct {
	tabPage    *walk.TabPage
	tabIndex   int
	tabTitle   string
	statusText string
}

func (t *tabViewCommon) Id() int            { return t.tabIndex }
func (t *tabViewCommon) StatusText() string { return t.statusText }
func (t *tabViewCommon) Title() string      { return t.tabTitle }

type tabViewWithInput interface {
	Send(string)
	Println(string)
	TabComplete(string) []string
}

type tabViewChatbox struct {
	tabViewCommon
	textBuffer *walk.TextEdit
	textInput  *MyLineEdit
}

func (cb *tabViewChatbox) Focus() {
	cb.tabPage.SetTitle(strings.TrimPrefix(cb.Title(), "* "))
}

func (cb *tabViewChatbox) Println(msg string) {
	mw.WindowBase.Synchronize(func() {
		cb.textBuffer.AppendText(msg + "\r\n")
		if cb.tabIndex != tabWidget.CurrentIndex() {
			if !strings.HasPrefix(cb.Title(), "* ") {
				cb.tabPage.SetTitle("* " + cb.Title())
			}
		}
	})
}

type tabViewServer struct {
	tabViewChatbox
}

func (t *tabViewServer) Send(search string) {}
func (t *tabViewServer) TabComplete(search string) []string {
	return []string{}
}

func NewServerTab(conn *serverConnection, serv *serverState) *tabViewServer {
	t := &tabViewServer{
		tabTitle:   serv.networkName,
		textBuffer: &walk.TextEdit{},
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)
		TextEdit{
			AssignTo:           &t.textBuffer,
			ReadOnly:           true,
			AlwaysConsumeSpace: true,
			Persistent:         true,
			VScroll:            true,
			MaxLength:          0x7FFFFFFE,
		}.Create(builder)
		textInput := NewTextInput(t, &clientContext{conn, serv.networkName, t})
		checkErr(t.tabPage.Children().Add(textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
	})
	return t
}

type tabViewChannel struct {
	tabViewChatbox
	topicInput       *walk.LineEdit
	nickList         *nickList
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	send             func(string)
}

func (t *tabViewChannel) Send(message string) {
	t.send(message)
}

func (t *tabViewChannel) TabComplete(search string) []string {
	return t.nickList.Search(search)
}

func NewChannelTab(conn *serverConnection, serv *serverState, channel *channelState) *tabViewChannel {
	t := &tabViewChannel{
		tabTitle:   channel.Channel,
		textBuffer: &walk.TextEdit{},
		nickList:   newNickList(),
	}
	t.send = func(msg string) {
		conn.Privmsg(channel.channel, msg)
		nick := t.nickList.Get(serv.User.Nick)
		t.Println(fmt.Sprintf("%s <%s> %s", now(), nick, msg))
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)
		TextEdit{
			AssignTo:           &t.textBuffer,
			ReadOnly:           true,
			AlwaysConsumeSpace: true,
			Persistent:         true,
			VScroll:            true,
			MaxLength:          0x7FFFFFFE,
		}.Create(builder)
		textInput := NewTextInput(t, &clientContext{conn, channel.channel, t})
		checkErr(t.tabPage.Children().Add(textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
	})
	return t
}

type tabViewPrivmsg struct {
	tabViewChatbox
}

func (t *tabViewPrivmsg) Send(message string) {
	t.send(message)
}
func (t *tabViewPrivmsg) TabComplete(search string) []string {
	return []string{}
}

func NewPrivmsgTab(conn *serverConnection, serv *serverState, privmsg *privmsgState) *tabViewPrivmsg {
	t := &tabViewPrivmsg{
		tabTitle:   privmsg.nick,
		textBuffer: &walk.TextEdit{},
	}
	t.send = func(msg string) {
		conn.Privmsg(privmsg.nick, msg)
		nick := newNick(serv.user.nick)
		t.Println(fmt.Sprintf("%s <%s> %s", now(), nick, msg))
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)
		TextEdit{
			AssignTo:           &t.textBuffer,
			ReadOnly:           true,
			AlwaysConsumeSpace: true,
			Persistent:         true,
			VScroll:            true,
			MaxLength:          0x7FFFFFFE,
		}.Create(builder)
		textInput := NewTextInput(t, &clientContext{conn, privmsg.nick, t})
		checkErr(t.tabPage.Children().Add(textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
	})
	return t
}
