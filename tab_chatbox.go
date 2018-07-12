package main

import (
	"fmt"

	"github.com/lxn/win"
)

type tabChatbox struct {
	tabCommon
	unread       int
	disconnected bool
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
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}
func (t *tabChatbox) Focus() {
	t.unread = 0
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
	})
	statusBar.SetText(t.statusText)
	t.textInput.SetFocus()
	t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
}

func (t *tabChatbox) Println(msg string) {
	t.chatlogger(msg)

	text, styles := parseString(msg)
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
