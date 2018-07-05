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
	tabView
	Send(string)
	Println(string)
	Clear()
}

type tabViewCommon struct {
	tabIndex   int
	tabTitle   string
	tabPage    *walk.TabPage
	statusText string
}

func (t *tabViewCommon) Id() int {
	return tabWidget.Pages().Index(t.tabPage)
}
func (t *tabViewCommon) StatusText() string { return t.statusText }
func (t *tabViewCommon) HasFocus() bool {
	return t.tabIndex == tabWidget.CurrentIndex()
}
func (t *tabViewCommon) Close() {
	// NOTE(tso): TODO
}

type tabViewChatbox struct {
	tabViewCommon
	unread       int
	disconnected bool
	textBuffer   *walk.TextEdit
	textInput    *MyLineEdit
}

func (t *tabViewChatbox) Clear() {
	t.textBuffer.SetText("")
}

func (t *tabViewChatbox) Title() string {
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
func (t *tabViewChatbox) Focus() {
	t.unread = 0
	t.tabPage.SetTitle(t.Title())
	statusBar.SetText(t.statusText)
	t.textInput.SetFocus()
}

func (t *tabViewChatbox) Println(msg string) {
	mw.WindowBase.Synchronize(func() {
		t.textBuffer.AppendText(msg + "\r\n")
		if !t.HasFocus() {
			t.unread++
			t.tabPage.SetTitle(t.Title())
		}
	})
}

type tabViewServer struct {
	tabViewChatbox
}

// func Errorln() ???

func (t *tabViewServer) Send(search string) {}

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

func NewServerTab(servConn *serverConnection, servState *serverState) *tabViewServer {
	t := &tabViewServer{}
	t.tabTitle = servState.networkName
	t.textBuffer = &walk.TextEdit{}

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
		t.textInput = NewTextInput(t, &clientContext{
			servConn:  servConn,
			channel:   servState.networkName,
			cb:        t,
			servState: servState,
			chanState: nil,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		tabWidget.SaveState()
		tabs = append(tabs, t)
		checkErr(tabWidget.SetCurrentIndex(index))
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
	tabViewChatbox
	topicInput       *walk.LineEdit
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	send             func(string)
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

func NewChannelTab(servConn *serverConnection, servState *serverState, chanState *channelState) *tabViewChannel {
	t := &tabViewChannel{}
	t.tabTitle = chanState.channel
	t.textBuffer = &walk.TextEdit{}
	chanState.nickList = newNickList()
	t.nickListBox = &walk.ListBox{}
	t.nickListBoxModel = &listBoxModel{}
	t.topicInput = &walk.LineEdit{}
	t.send = func(msg string) {
		servConn.conn.Privmsg(chanState.channel, msg)
		nick := chanState.nickList.Get(servState.user.nick)
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
					OnItemActivated: func() {
						nick := newNick(t.nickListBoxModel.Items[t.nickListBox.CurrentIndex()])

						pmState, ok := servState.privmsgs[nick.name]
						if !ok {
							pmState = &privmsgState{
								nick: nick.name,
							}
							pmState.tab = NewPrivmsgTab(servConn, servState, pmState)
							servState.privmsgs[nick.name] = pmState
						}
						mw.WindowBase.Synchronize(func() {
							checkErr(tabWidget.SetCurrentIndex(pmState.tab.Id()))
						})
					},
				},
			},
			AlwaysConsumeSpace: true,
		}.Create(builder)

		checkErr(hsplit.SetHandleWidth(1))

		t.textInput = NewTextInput(t, &clientContext{
			servConn:  servConn,
			channel:   chanState.channel,
			cb:        t,
			servState: servState,
			chanState: chanState,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		tabWidget.SaveState()
		tabs = append(tabs, t)
		checkErr(tabWidget.SetCurrentIndex(index))
	})
	return t
}

type tabViewPrivmsg struct {
	tabViewChatbox
	send func(string)
}

func (t *tabViewPrivmsg) Send(message string) {
	t.send(message)
}

func (t *tabViewPrivmsg) Update(servState *serverState, pmState *privmsgState) {
	t.statusText = servState.tab.statusText
	if t.HasFocus() {
		statusBar.SetText(t.statusText)
	}
}

func NewPrivmsgTab(servConn *serverConnection, servState *serverState, pmState *privmsgState) *tabViewPrivmsg {
	t := &tabViewPrivmsg{}
	t.tabTitle = pmState.nick
	t.textBuffer = &walk.TextEdit{}
	t.send = func(msg string) {
		servConn.conn.Privmsg(pmState.nick, msg)
		nick := newNick(servState.user.nick)
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
		t.textInput = NewTextInput(t, &clientContext{
			servConn:  servConn,
			channel:   pmState.nick,
			cb:        t,
			servState: servState,
			chanState: nil,
			pmState:   pmState,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		t.tabIndex = index
		tabs = append(tabs, t)
		tabWidget.SaveState()
		// NOTE(tso): don't steal focus
		// checkErr(tabWidget.SetCurrentIndex(index))
	})
	return t
}
