package main

import (
	"errors"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

func SetStatusBarIcon(icofile string) {
	if icofile == "" {
		return
	}

	// ico, err := walk.NewIconFromFile(icofile)
	// checkErr(err)
	// statusBar.SetIcon(ico)
	// lol jk that's too easy hold my beer

	absFilePath, err := filepath.Abs(icofile)
	checkErr(err)
	hIcon := win.HICON(win.LoadImage(
		0,
		syscall.StringToUTF16Ptr(absFilePath),
		win.IMAGE_ICON,
		16, // size x
		16, // size y
		win.LR_LOADFROMFILE))

	if hIcon == 0 {
		checkErr(errors.New("LoadImage failed to load image: " + absFilePath))
	}

	mw.StatusBar().SendMessage(win.SB_SETICON, 0, uintptr(hIcon))
}

func SetStatusBarText(text string) {
	txt, err := syscall.UTF16PtrFromString(text)
	checkErr(err)
	mw.StatusBar().SendMessage(win.SB_SETTEXT, win.SBT_OWNERDRAW, uintptr(unsafe.Pointer(txt)))
}
