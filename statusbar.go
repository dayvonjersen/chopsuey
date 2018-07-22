package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

func SetStatusBarText(text string) {
	txt, err := syscall.UTF16PtrFromString(text)
	checkErr(err)
	mw.StatusBar().SendMessage(win.SB_SETTEXT, win.SBT_OWNERDRAW, uintptr(unsafe.Pointer(txt)))
}
