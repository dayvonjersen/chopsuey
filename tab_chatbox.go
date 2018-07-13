package main

import (
	"fmt"

	"github.com/lxn/win"
)

type tabChatbox struct {
	tabCommon
	unread       int
	disconnected bool
	error        bool
	notify       bool
	textBuffer   *RichEdit
	textInput    *MyLineEdit
	chatlogger   func(string)
}

func (t *tabChatbox) Clear() {
	t.textBuffer.SetText("")
}

func (t *tabChatbox) Title() string {
	title := t.tabTitle
	// add nickflash here
	if t.unread > 0 && !t.HasFocus() {
		title = fmt.Sprintf("%s [%d]", title, t.unread)
	}
	if t.notify {
		title = " * " + title
	}
	if t.error {
		title = "/!\\ " + title
	}
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}

func (t *tabChatbox) Notify() {
	t.notify = true
}

func (t *tabChatbox) Focus() {
	t.unread = 0
	t.error = false
	t.notify = false
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
	})
	statusBar.SetText(t.statusText)
	t.textInput.SetFocus()
	t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
}

func (t *tabChatbox) Logln(text string) {
	t.chatlogger(text)
}

func (t *tabChatbox) Errorln(text string, styles [][]int) {
	t.error = true
	statusBar.SetText(text)
	// TODO(tso): set status bar icon
	t.Println(text, styles)
}

func (t *tabChatbox) Println(text string, styles [][]int) {
	mw.WindowBase.Synchronize(func() {
		t.textBuffer.AppendText("\n")

		// HACK(tso): shouldn't have to clear styles like this
		if t.textBuffer.linecount > 1 {
			l := t.textBuffer.TextLength()
			t.textBuffer.ResetText(l-t.textBuffer.linecount, l-t.textBuffer.linecount)
		}

		t.textBuffer.AppendText(text, styles...)

		// HACK(tso): and we shouldn't have to do it twice
		l := t.textBuffer.TextLength()
		t.textBuffer.ResetText(l-t.textBuffer.linecount, l-t.textBuffer.linecount)

		if !t.HasFocus() {
			t.unread++
			t.tabPage.SetTitle(t.Title())
		}

		if t.textInput.Focused() || !mainWindowFocused {
			t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
		}
	})
}