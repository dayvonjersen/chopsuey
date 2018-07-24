package main

import (
	"fmt"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type tabServer struct {
	tabChatbox
}

// FIXME(tso): having these here just to satisfy the interface seems wrong.
func (t *tabServer) NickColor(nick string) int { return Black }
func (t *tabServer) Send(message string)       {}

func (t *tabServer) Update(servState *serverState) {
	if t.tabTitle != servState.networkName {
		t.tabTitle = servState.networkName
	}

	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
		if t.HasFocus() {
			SetStatusBarIcon(t.statusIcon)
			SetStatusBarText(t.statusText)
		}
	})

	switch servState.connState {
	case CONNECTION_EMPTY:
		t.disconnected = true
		t.statusIcon = "res/conn_pcs_no_network.ico"
		t.statusText = "not connected to any network"
	case DISCONNECTED:
		t.disconnected = true
		t.statusIcon = "res/conn_pcs_no_network.ico"
		t.statusText = "disconnected x_x"
		Println(CLIENT_ERROR, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTING:
		t.disconnected = true
		t.statusIcon = "res/conn_pcs_off_on.ico"
		t.statusText = "connecting to " + servState.networkName + "..."
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTION_ERROR:
		t.disconnected = true
		t.statusIcon = "res/conn_pcs_no_network.ico"
		t.statusText = "couldn't connect: " + servState.lastError.Error()
		Println(CLIENT_ERROR, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTION_START:
		t.disconnected = false
		t.statusIcon = "res/conn_pcs_on_off.ico"
		t.statusText = "connected to " + servState.networkName
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTED:
		t.statusIcon = "res/network_three_pcs.ico"
		t.statusText = fmt.Sprintf("%s connected to %s", servState.user.nick, servState.networkName)
		t.disconnected = false
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
	SetSystrayContextMenu()
}

func NewServerTab(servConn *serverConnection, servState *serverState) *tabServer {
	t := &tabServer{}
	clientState.AppendTab(t)
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
		t.textBuffer.KeyPress().Attach(ctrlTab)
		t.textInput = NewTextInput(t, &commandContext{
			servConn:  servConn,
			tab:       t,
			servState: servState,
			chanState: nil,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))

		// remove borders
		win.SetWindowLong(t.textInput.Handle(), win.GWL_EXSTYLE, 0)

		checkErr(tabWidget.Pages().Add(t.tabPage))
		index := tabWidget.Pages().Index(t.tabPage)
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
		t.Focus()

		applyThemeToTab(t)
	})
	return t
}
