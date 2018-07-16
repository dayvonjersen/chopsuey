package main

import (
	"fmt"

	"github.com/lxn/walk"
	"github.com/lxn/win"
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
		if t.HasFocus() {
			statusBar.SetText(t.statusText)
		}
	})

	switch servState.connState {
	case CONNECTION_EMPTY:
		t.disconnected = true
		t.statusText = "not connected to any network"
	case DISCONNECTED:
		t.disconnected = true
		t.statusText = "disconnected x_x"
		Println(CLIENT_ERROR, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTING:
		t.disconnected = true
		t.statusText = "connecting to " + servState.networkName + "..."
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTION_ERROR:
		t.disconnected = true
		t.statusText = "couldn't connect: " + servState.lastError.Error()
		Println(CLIENT_ERROR, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTION_START:
		t.disconnected = false
		t.statusText = "connected to " + servState.networkName
		Println(CLIENT_MESSAGE, T(servState.AllTabs()...), now(), t.statusText)
	case CONNECTED:
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

		bg := globalBackgroundColor
		bgColorref := uint32(bg&0xff<<16 | bg&0xff00 | bg&0xff0000>>16)
		t.textBuffer.SendMessage(win.WM_USER+67, 0, uintptr(bgColorref))
		/*		brush, err := walk.NewSolidColorBrush(walk.RGB(r, g, b))
				checkErr(err)
				//defer brush.Dispose()
				mw.SetBackground(brush)
				fg := globalForegroundColor
				colorref := win.COLORREF(fg&0xff<<16 | fg&0xff00 | fg&0xff0000>>16)
				win.SetTextColor(win.GetDC(re.Handle()), colorref)
		*/
	})
	return t
}
