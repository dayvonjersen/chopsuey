package main

import (
	"github.com/lxn/walk"
)

type tab interface {
	Index() int
	Title() string
	StatusIcon() string
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

	Padlen(string) int    // just shoehorning this in here for now
	NickColor(string) int // ditto

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
	statusIcon string
	statusText string
}

func (t *tabCommon) Index() int {
	return tabWidget.Pages().Index(t.tabPage)
}

func (t *tabCommon) StatusIcon() string {
	return t.statusIcon
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

	// for when we implement closing tabs in ways other than /close
	shouldChangeTabFocus := t.HasFocus()
	myIndexWas := t.Index()

	tabMan.Delete(t)

	checkErr(tabWidget.Pages().Remove(t.tabPage))
	t.tabPage.Dispose()
	tabWidget.SaveState()

	if tabWidget.Pages().Len() == 0 {
		tabWidget.Pages().Clear()
	}
	if shouldChangeTabFocus {
		newIndex := myIndexWas - 1
		if newIndex < 0 {
			newIndex = 0
		}
		checkErr(tabWidget.SetCurrentIndex(newIndex))

		tabWidget.SaveState()
	} else {
		tabWidget.SetCurrentIndex(tabWidget.CurrentIndex())
	}
}
