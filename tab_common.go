package main

import (
	"github.com/lxn/walk"
)

type tab interface {
	Index() int
	Title() string
	StatusText() string
	HasFocus() bool
	Focus()
	Close()
}

type tabWithInput interface {
	tab // inherit all from above

	Send(string) // send to channel/nick
}

type tabWithTextBuffer interface {
	tab

	Logln(string)            // chatlogging
	Errorln(string, [][]int) // print error to buffer
	Println(string, [][]int) // print text to buffer
	// TODO(tso): better name for Notify/t.notify/asterisk what are words
	Notify(bool) // put a * in the tab title

	Clear() // clear buffer
}

type tabCommon struct {
	tabTitle   string
	tabPage    *walk.TabPage
	statusText string
}

func (t *tabCommon) Index() int {
	return tabWidget.Pages().Index(t.tabPage)
}

func (t *tabCommon) StatusText() string {
	return t.statusText
}

func (t *tabCommon) Title() string { return "##################" }
func (t *tabCommon) Focus()        {}

func (t *tabCommon) HasFocus() bool {
	return mainWindowFocused && t.Index() == tabWidget.CurrentIndex()
}

func (t *tabCommon) Close() {
	clientState.RemoveTab(t)
	mw.WindowBase.Synchronize(func() {
		mw.WindowBase.SetSuspended(true)
		defer mw.WindowBase.SetSuspended(false)

		checkErr(tabWidget.Pages().Remove(t.tabPage))
		t.tabPage.Dispose()
		tabWidget.SaveState()

		if tabWidget.Pages().Len() > 0 {
			checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Len() - 1))
		} else {
			tabWidget.Pages().Clear()
		}
		tabWidget.SaveState()
	})
}
