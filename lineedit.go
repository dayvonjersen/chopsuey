package main

import (
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
