package main

import (
	"fmt"
	"log"
	"unsafe"

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
	if t.textBuffer == nil {
		return
	}
	lpsi := win.SCROLLINFO{}
	lpsi.FMask = win.SIF_ALL
	lpsi.CbSize = uint32(unsafe.Sizeof(lpsi))
	shouldScroll := false
	if win.GetScrollInfo(t.textBuffer.Handle(), win.SB_VERT, &lpsi) {
		min := int(lpsi.NMin)
		max := int(lpsi.NMax)
		pos := int(int32(lpsi.NPage) + lpsi.NPos)
		log.Printf("lpsi: %v min: %v max: %v pos: %v", lpsi, min, max, pos)
		if lpsi.NPage == 0 {
			shouldScroll = true
		} else {
			shouldScroll = pos >= max
		}
	} else {
		log.Println("failed to GetScrollInfo()!")
	}
	if t.unread > 0 && !t.unreadSpaced {
		// TODO(tso): think of something better than a bunch of whitespace
		//            because apparently I have a tendency to focus and unfocus
		//            the window without thinking about it
		//
		//            and chat
		//
		//            ends up
		//
		//            looking
		//
		//            like this
		//
		//            maybe put an arrow instead of the timestamp where the first new message begins
		// -tso 7/15/2018 1:07:16 PM
		t.unreadSpaced = true
	}

	t.textBuffer.AppendText("\n")
	t.textBuffer.AppendText(text, styles...)

	if shouldScroll {
		t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
	}

}
