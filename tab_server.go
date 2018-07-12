package main

import (
	"fmt"

	"github.com/lxn/walk"
)

type tabServer struct {
	tabChatbox
}

// func Errorln() ???

func (t *tabServer) Send(message string) {
	// NOTE(tso): idea: send raw commands in the server tab e.g.
	// PRIVMSG #go-nuts :hi guys
}

func (t *tabServer) Update(servState *serverState) {
	if t.tabTitle != servState.networkName {
		t.tabTitle = servState.networkName
	}
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
	})

	switch servState.connState {
	case CONNECTION_EMPTY:
		t.statusText = "not connected to any network"
		t.disconnected = true
	case DISCONNECTED:
		t.statusText = "disconnected x_x"
		t.Println(now() + " " + t.statusText)
		t.disconnected = true
	case CONNECTING:
		t.statusText = "connecting to " + servState.networkName + "..."
		t.Println(now() + " " + t.statusText)
		t.disconnected = true
	case CONNECTION_ERROR:
		t.statusText = "couldn't connect: " + servState.lastError.Error()
		t.Println(now() + " ERROR: " + t.statusText)
		t.disconnected = true
	case CONNECTION_START:
		t.statusText = "connected to " + servState.networkName
		t.disconnected = false
	case CONNECTED:
		t.statusText = fmt.Sprintf("%s connected to %s", servState.user.nick, servState.networkName)
		t.disconnected = false
	}
	if t.HasFocus() {
		statusBar.SetText(t.statusText)
	}
	for _, chanState := range servState.channels {
		chanState.tab.Update(servState, chanState)
	}
	for _, pmState := range servState.privmsgs {
		pmState.tab.Update(servState, pmState)
	}
	if servState.channelList != nil {
		servState.channelList.Update(servState)
	}
}

func NewServerTab(servConn *serverConnection, servState *serverState) *tabServer {
	t := &tabServer{}
	t.tabTitle = servState.networkName
	t.chatlogger = NewChatLogger(servState.networkName)

	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		t.textBuffer, err = NewRichEdit(t.tabPage)
		checkErr(err)
		t.textInput = NewTextInput(t, &commandContext{
			servConn:  servConn,
			tab:       t,
			servState: servState,
			chanState: nil,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))

		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
		t.Focus()
		tabs = append(tabs, t)
	})
	return t
}
