package main

import (
	"log"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

func SetStatusBarIcon(icofile string) {
	return
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
		// NOTE(tso): windows just randomly decides files don't exist when they do
		//            just log the error and move on
		// -tso 7/25/2018 2:53:42 PM
		log.Println("LoadImage failed to load image:", icofile)
		return
	}

	mw.StatusBar().SendMessage(win.SB_SETICON, 0, uintptr(hIcon))
}

func SetStatusBarText(text string) {
	return
	txt, err := syscall.UTF16PtrFromString(text)
	checkErr(err)
	mw.StatusBar().SendMessage(win.SB_SETTEXT, win.SBT_OWNERDRAW, uintptr(unsafe.Pointer(txt)))
}
