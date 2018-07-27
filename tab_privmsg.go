package main

import (
	"math/rand"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type tabPrivmsg struct {
	tabChatbox
	send      func(string)
	nickColor func(string) int
}

func (t *tabPrivmsg) NickColor(nick string) int {
	return t.nickColor(nick)
}

func (t *tabPrivmsg) Send(message string) {
	t.send(message)
}

func (t *tabPrivmsg) Update(servState *serverState, pmState *privmsgState) {
	t.disconnected = servState.connState != CONNECTED
	t.statusIcon = servState.tab.statusIcon
	t.statusText = servState.tab.statusText
	if t.tabPage != nil {
		mw.WindowBase.Synchronize(func() {
			t.tabPage.SetTitle(t.Title())
			if t.HasFocus() {
				SetStatusBarIcon(t.statusIcon)
				SetStatusBarText(t.statusText)
			}
		})
	}

	SetSystrayContextMenu()
}

func newPrivmsgTab(servConn *serverConnection, servState *serverState, pmState *privmsgState, tabIndex int) *tabPrivmsg {
	t := &tabPrivmsg{}
	t.tabTitle = pmState.nick

	t.send = func(msg string) {
		servConn.conn.Privmsg(pmState.nick, msg)
		nick := newNick(servState.user.nick)
		privateMessage(t, nick.String(), msg)
	}

	color := rand.Intn(98)
	t.nickColor = func(nick string) int {
		if nick == servState.user.nick {
			return DarkGray
		}
		return color
	}

	t.chatlogger = NewChatLogger(servState.networkName + "-" + pmState.nick)

	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		t.textBuffer, err = NewRichEdit(t.tabPage)
		checkErr(err)
		// WTF(tso): textBuffer (*RichEdit) is already attached
		//           to t.tabPage (*walk.TabPage) because of walk.InitWidget but that
		//           *doesn't happen* when you use the walk/declarative interface
		// wtf -tso 7/12/2018 1:54:43 AM
		// checkErr(t.tabPage.Children().Add(t.textBuffer))
		t.textInput = NewTextInput(t)
		checkErr(t.tabPage.Children().Add(t.textInput))

		// remove borders
		win.SetWindowLong(t.textInput.Handle(), win.GWL_EXSTYLE, 0)

		{
			index := servState.tab.Index()
			if servState.channelList != nil {
				index = servState.channelList.Index()
			}
			for _, ch := range servState.channels {
				i := ch.tab.Index()
				if i > index {
					index = i
				}
			}
			for _, pm := range servState.privmsgs {
				i := pm.tab.Index()
				if i > index {
					index = i
				}
			}
			index++

			checkErr(tabWidget.Pages().Insert(index, t.tabPage))
		}

		// NOTE(tso): don't steal focus
		// index := tabWidget.Pages().Index(t.tabPage)
		// checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
		applyThemeToTab(t)
		pmState.tab = t
		servState.privmsgs[pmState.nick] = pmState
		servState.tab.Update(servState)
	})

	return t
}
