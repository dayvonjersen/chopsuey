package main

import (
	"fmt"

	"github.com/lxn/win"
)

type tabChatbox struct {
	tabCommon
	unread       int
	unreadSpaced bool
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
	if t.unread > 0 && !t.HasFocus() {
		title = fmt.Sprintf("%s %d", title, t.unread)
	}
	if t.notify {
		title = "* " + title
	}
	if t.error {
		title = "! " + title
	}
	if t.disconnected {
		title = "(" + title + ")"
	}
	return title
}

// TODO(tso): think of a better name than UpdateMessageCounterAndPossiblyNickFlashSlashHighlight
func (t *tabChatbox) Notify(asterisk bool) {
	if !t.HasFocus() {
		if asterisk {
			t.notify = true
		}
		t.unread++
		mw.WindowBase.Synchronize(func() {
			t.tabPage.SetTitle(t.Title())
		})
	}
}

func (t *tabChatbox) Focus() {
	t.unread = 0
	t.unreadSpaced = false
	t.error = false
	t.notify = false
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
		statusBar.SetText(t.statusText)
		t.textInput.SetFocus()
		t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
	})
}

func (t *tabChatbox) Logln(text string) {
	t.chatlogger(text)
}

func (t *tabChatbox) Errorln(text string, styles [][]int) {
	if !t.HasFocus() {
		t.error = true
	}
	mw.WindowBase.Synchronize(func() {
		statusBar.SetText(text)
	})
	// TODO(tso): set status bar icon
	t.Println(text, styles)
}

func (t *tabChatbox) Println(text string, styles [][]int) {
	mw.WindowBase.Synchronize(func() {
		if t.unread > 0 && !t.unreadSpaced {
			t.textBuffer.AppendText("\n\n\n- - - - - - - - - - - - - - - - - - - - - - -") // TODO(tso): think of something better
			t.unreadSpaced = true
		}

		t.textBuffer.AppendText("\n")
		t.textBuffer.AppendText(text, styles...)

		if t.textInput.Focused() || !mainWindowFocused {
			t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
		}
	})
}
