package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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
	tab
	Send(string)
	Println(string)
	Clear()
}

type tabCommon struct {
	tabTitle   string
	tabPage    *walk.TabPage
	statusText string
}

func (t *tabCommon) Index() int {
	return tabWidget.Pages().Index(t.tabPage)
}
func (t *tabCommon) StatusText() string { return t.statusText }
func (t *tabCommon) HasFocus() bool {
	return mainWindowFocused && t.Index() == tabWidget.CurrentIndex()
}
func (t *tabCommon) Close() {
	index := t.Index()
	for i, tab := range tabs {
		if tab.Index() == index {
			tabs = append(tabs[0:i], tabs[i+1:]...)
			break
		}
	}
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
