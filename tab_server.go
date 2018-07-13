package main

import (
	"fmt"
	"log"

	"github.com/lxn/walk"
)

type tabServer struct {
	tabChatbox
}

func (t *tabServer) Send(message string) {}

func (t *tabServer) Update(servState *serverState) {
	if t.tabTitle != servState.networkName {
		t.tabTitle = servState.networkName
	}

	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
	})

	switch servState.connState {
	case CONNECTION_EMPTY:
		t.disconnected = true
		t.statusText = "not connected to any network"
	case DISCONNECTED:
		t.disconnected = true
		t.statusText = "disconnected x_x"
		Println(CLIENT_ERROR, T(servState.AllTabs()...), t.statusText)
	case CONNECTING:
		t.disconnected = true
		t.statusText = "connecting to " + servState.networkName + "..."
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), t.statusText)
	case CONNECTION_ERROR:
		t.disconnected = true
		t.statusText = "couldn't connect: " + servState.lastError.Error()
		Println(CLIENT_ERROR, T(servState.AllTabs()...), t.statusText)
	case CONNECTION_START:
		t.disconnected = false
		t.statusText = "connected to " + servState.networkName
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), t.statusText)
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
	tabs = append(tabs, t)
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

		log.Printf("tabWidget: %v", tabWidget)
		log.Printf("tabWidget.Pages(): %v", tabWidget.Pages())
		log.Printf("tabPage: %v", t.tabPage)
		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
		t.Focus()
	})
	return t
}