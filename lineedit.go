package main

import (
	"strings"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

func newMyLineEdit(parent walk.Container) *MyLineEdit {
	le := new(MyLineEdit)
	checkErr(walk.InitWindow(
		le,
		parent,
		"EDIT",
		win.WS_CHILD|win.WS_TABSTOP|win.WS_VISIBLE|win.ES_AUTOHSCROLL,
		win.WS_EX_CLIENTEDGE,
	))
	return le
}

type MyLineEdit struct {
	walk.LineEdit

	msgHistory      []string
	msgHistoryIndex int
	tabComplete     *tabComplete
}

type tabComplete struct {
	Active  bool
	Entries []string
	Index   int
}

// http://forums.codeguru.com/showthread.php?60142-How-to-trap-WM_KILLFOCUS-with-Tab-key
func (le *MyLineEdit) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_GETDLGCODE {
		if wParam == win.VK_TAB || wParam == win.VK_ESCAPE || wParam == win.VK_RETURN {
			return win.DLGC_WANTMESSAGE
		}
	}
	return le.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

func NewTextInput(t *tabViewWithInput, ctx *clientContext) *MyLineEdit {
	textInput := newMyLineEdit(t.tabPage)

	textInput.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			text := textInput.Text()
			if len(text) < 1 {
				return
			}
			textInput.msgHistory = append(textInput.msgHistory, text)
			textInput.msgHistoryIdx = len(textInput.msgHistory) - 1
			if text[0] == '/' {
				parts := strings.Split(text[1:], " ")
				cmd := parts[0]
				if cmd[0] == '/' {
					t.Send(cmd)
				} else {
					var args []string
					if len(parts) > 1 {
						args = parts[1:]
					} else {
						args = []string{}
					}
					if cmdFn, ok := clientCommands[cmd]; ok {
						cmdFn(ctx, args...)
					} else {
						t.Println("unrecognized command: " + cmd)
					}
				}
			} else {
				t.Send(text)
			}
			textInput.SetText("")
		} else if key == walk.KeyUp {
			if len(textInput.msgHistory) > 0 {
				text := textInput.msgHistory[textInput.msgHistoryIdx]
				textInput.SetText(text)
				textInput.SetTextSelection(len(text), len(text))
				textInput.msgHistoryIdx--
				if textInput.msgHistoryIdx < 0 {
					textInput.msgHistoryIdx = 0
				}
			}
		} else if key == walk.KeyDown {
			if len(textInput.msgHistory) > 0 {
				textInput.msgHistoryIdx++
				if textInput.msgHistoryIdx <= len(textInput.msgHistory)-1 {
					text := textInput.msgHistory[textInput.msgHistoryIdx]
					textInput.SetText(text)
					textInput.SetTextSelection(len(text), len(text))
				} else {
					textInput.SetText("")
					textInput.msgHistoryIdx = len(textInput.msgHistory) - 1
				}
			}
		}
	})

	textInput.KeyUp().Attach(func(key walk.Key) {
		if key == walk.KeyUp || key == walk.KeyDown {
			text := textInput.Text()
			textInput.SetTextSelection(len(text), len(text))
		}
	})

	textInput.KeyPress().Attach(func(key walk.Key) {
		if key == walk.KeyUp || key == walk.KeyDown {
			text := textInput.Text()
			textInput.SetTextSelection(len(text), len(text))
		} else if key == walk.KeyTab {
			text := strings.Split(textInput.Text(), " ")
			if textInput.tabComplete.Active {
				textInput.tabComplete.Index++
				if textInput.tabComplete.Index >= len(textInput.tabComplete.Entries) {
					textInput.tabComplete.Index = 0
				}
			} else {
				term := text[len(text)-1]
				res := t.TabComplete(term)
				res = append(res, term)
				textInput.tabComplete = &tabComplete{
					Active:  true,
					Entries: res,
					Index:   0,
				}
			}
			text = append(text[:len(text)-1], textInput.tabComplete.Entries[textInput.tabComplete.Index])
			txt := strings.Join(text, " ")
			textInput.SetText(txt)
			textInput.SetTextSelection(len(txt), len(txt))
		} else {
			if textInput.tabComplete.Active {
				textInput.tabComplete = &tabComplete{}
			}
		}
	})
	return textInput
}
