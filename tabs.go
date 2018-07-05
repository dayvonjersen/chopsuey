package main

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type tabView interface {
	Id() int
	Title() string
	StatusText() string
	HasFocus() bool
	Focus()
	Close()
}

type tabViewWithInput interface {
	Send(string)
	Println(string)
}

type tabViewServer struct {
	tabPage      *walk.TabPage
	tabIndex     int
	tabTitle     string
	statusText   string
	unread       int
	disconnected bool
	textBuffer   *walk.TextEdit
	textInput    *MyLineEdit
}

func (t *tabViewServer) Id() int            { return t.tabIndex }
func (t *tabViewServer) StatusText() string { return t.statusText }
func (t *tabViewServer) Title() string {
	title := t.tabTitle
	// add nickflash here
	if t.unread > 0 && !t.HasFocus() {
		title = fmt.Sprintf("%s [%d]", title, t.unread)
	}
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}
func (t *tabViewServer) HasFocus() bool {
	return t.tabIndex == tabWidget.CurrentIndex()
}
func (cb *tabViewServer) Focus() {
	cb.unread = 0
	cb.tabPage.SetTitle(cb.Title())
}

// func Errorln() ???

func (t *tabViewServer) Println(msg string) {
	mw.WindowBase.Synchronize(func() {
		t.textBuffer.AppendText(msg + "\r\n")
		if !t.HasFocus() {
			t.unread++
			t.tabPage.SetTitle(t.Title())
		}
	})
}

func (t *tabViewServer) Send(search string) {}
func (t *tabViewServer) TabComplete(search string) []string {
	return []string{}
}

func (t *tabViewServer) Update(servState *serverState) {
	if t.tabTitle != servState.networkName {
		t.tabTitle = servState.networkName
	}
	t.tabPage.SetTitle(t.Title())

	if servState.connected {
		t.statusText = fmt.Sprintf("%s connected to %s", servState.user.nick, servState.networkName)
	} else {
		t.statusText = "disconnected x_x"
	}
	for _, chanState := range servState.channels {
		chanState.tab.Update(servState, chanState)
	}
	for _, pmState := range servState.privmsgs {
		pmState.tab.Update(servState, pmState)
	}
	if t.HasFocus() {
		statusBar.SetText(t.statusText)
	}
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
		textInput := NewTextInput(t, &clientContext{
			servConn:     conn,
			channel:      serv.networkName,
			cb:           t,
			serverState:  serv,
			channelState: nil,
			privmsgState: nil,
		})
		checkErr(t.tabPage.Children().Add(textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
	})
	return t
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

type tabViewChannel struct {
	tabPage          *walk.TabPage
	tabIndex         int
	tabTitle         string
	statusText       string
	unread           int
	disconnected     bool
	textBuffer       *walk.TextEdit
	textInput        *MyLineEdit
	topicInput       *walk.LineEdit
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	send             func(string)
}

func (t *tabViewChannel) Id() int            { return t.tabIndex }
func (t *tabViewChannel) StatusText() string { return t.statusText }
func (t *tabViewChannel) Title() string {
	title := t.tabTitle
	// add nickflash here
	if t.unread > 0 && !t.HasFocus() {
		title = fmt.Sprintf("%s [%d]", title, t.unread)
	}
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}
func (t *tabViewChannel) HasFocus() bool {
	return t.tabIndex == tabWidget.CurrentIndex()
}

func (cb *tabViewChannel) Focus() {
	cb.unread = 0
	cb.tabPage.SetTitle(cb.Title())
}

// func Errorln() ???

func (t *tabViewChannel) Println(msg string) {
	mw.WindowBase.Synchronize(func() {
		t.textBuffer.AppendText(msg + "\r\n")
		if !t.HasFocus() {
			t.unread++
			t.tabPage.SetTitle(t.Title())
		}
	})
}

func (t *tabViewChannel) Send(message string) {
	t.send(message)
}

func (t *tabViewChannel) Update(servState *serverState, chanState *channelState) {
	t.statusText = servState.tab.statusText
	if t.HasFocus() {
		statusBar.SetText(t.statusText)
	}
}

func (t *tabViewChannel) updateNickList(chanState *channelState) {
	mw.WindowBase.Synchronize(func() {
		t.nickListBoxModel.Items = chanState.nickList.StringSlice()
		t.nickListBoxModel.PublishItemsReset()
	})
}

func NewChannelTab(conn *serverConnection, serv *serverState, channel *channelState) *tabViewChannel {
	t := &tabViewChannel{
		tabTitle:         channel.channel,
		textBuffer:       &walk.TextEdit{},
		nickListBox:      &walk.ListBox{},
		nickListBoxModel: &listBoxModel{},
		topicInput:       &walk.LineEdit{},
	}
	channel.nickList = newNickList()
	t.send = func(msg string) {
		conn.conn.Privmsg(channel.channel, msg)
		nick := channel.nickList.Get(serv.user.nick)
		t.Println(fmt.Sprintf("%s <%s> %s", now(), nick, msg))
	}
	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)

		LineEdit{
			AssignTo: &t.topicInput,
			ReadOnly: true,
		}.Create(builder)
		var hsplit *walk.Splitter
		HSplitter{
			AssignTo: &hsplit,
			Children: []Widget{
				TextEdit{
					AssignTo:           &t.textBuffer,
					ReadOnly:           true,
					AlwaysConsumeSpace: true,
					VScroll:            true,
					MaxLength:          0x7FFFFFFE,
					StretchFactor:      3,
				},
				ListBox{
					StretchFactor:      1,
					AssignTo:           &t.nickListBox,
					Model:              t.nickListBoxModel,
					AlwaysConsumeSpace: false,
					/*
						OnItemActivated: func() {
							nick := newNick(t.nickListBoxModel.Items[t.nickListBox.CurrentIndex()])
							box := conn.getChatBox(nick.name)
							if box == nil {
								cb.servConn.createChatBox(nick.name, CHATBOX_PRIVMSG)
							} else {
								checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(box.tabPage)))
							}
						},
					*/
				},
			},
			AlwaysConsumeSpace: true,
		}.Create(builder)
		checkErr(hsplit.SetHandleWidth(1))

		textInput := NewTextInput(t, &clientContext{
			servConn:     conn,
			channel:      channel.channel,
			cb:           t,
			serverState:  serv,
			channelState: channel,
			privmsgState: nil,
		})
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
	tabPage      *walk.TabPage
	tabIndex     int
	tabTitle     string
	statusText   string
	unread       int
	disconnected bool
	textBuffer   *walk.TextEdit
	textInput    *MyLineEdit
	send         func(string)
}

func (t *tabViewPrivmsg) Id() int            { return t.tabIndex }
func (t *tabViewPrivmsg) StatusText() string { return t.statusText }
func (t *tabViewPrivmsg) Title() string {
	title := t.tabTitle
	// add nickflash here
	if t.unread > 0 && !t.HasFocus() {
		title = fmt.Sprintf("%s [%d]", title, t.unread)
	}
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}
func (t *tabViewPrivmsg) HasFocus() bool {
	return t.tabIndex == tabWidget.CurrentIndex()
}

func (cb *tabViewPrivmsg) Focus() {
	cb.unread = 0
	cb.tabPage.SetTitle(cb.Title())
}

// func Errorln() ???

func (t *tabViewPrivmsg) Println(msg string) {
	mw.WindowBase.Synchronize(func() {
		t.textBuffer.AppendText(msg + "\r\n")
		if !t.HasFocus() {
			t.unread++
			t.tabPage.SetTitle(t.Title())
		}
	})
}

func (t *tabViewPrivmsg) Send(message string) {
	t.send(message)
}
func (t *tabViewPrivmsg) TabComplete(search string) []string {
	return []string{}
}
func (t *tabViewPrivmsg) Update(servState *serverState, pmState *privmsgState) {
	t.statusText = servState.tab.statusText
	if t.HasFocus() {
		statusBar.SetText(t.statusText)
	}
}

func NewPrivmsgTab(conn *serverConnection, serv *serverState, privmsg *privmsgState) *tabViewPrivmsg {
	t := &tabViewPrivmsg{
		tabTitle:   privmsg.nick,
		textBuffer: &walk.TextEdit{},
	}
	t.send = func(msg string) {
		conn.conn.Privmsg(privmsg.nick, msg)
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
		textInput := NewTextInput(t, &clientContext{
			servConn:     conn,
			channel:      privmsg.nick,
			cb:           t,
			serverState:  serv,
			channelState: nil,
			privmsgState: privmsg,
		})
		checkErr(t.tabPage.Children().Add(textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		// NOTE(tso): don't steal focus
		// checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
	})
	return t
}
