package main

import (
	"syscall"

	"github.com/lxn/win"
)

var (
	setLayeredWindowAttributes,
	showScrollBar uintptr
)

const (
	LWA_COLORKEY = 1
	LWA_ALPHA    = 2
)

func init() {
	libuser32 := win.MustLoadLibrary("user32.dll")
	setLayeredWindowAttributes = win.MustGetProcAddress(libuser32, "SetLayeredWindowAttributes")
	showScrollBar = win.MustGetProcAddress(libuser32, "ShowScrollBar")
}

func ShowScrollBar(hwnd win.HWND, wBar int, bShow int) {
	syscall.Syscall(showScrollBar, 3,
		uintptr(hwnd),
		uintptr(wBar),
		uintptr(bShow),
	)
}

func SetLayeredWindowAttributes(hwnd win.HWND, crKey, bAlpha, dwFlags int32) bool {
	ret, _, _ := syscall.Syscall6(setLayeredWindowAttributes, 4,
		uintptr(hwnd),
		uintptr(crKey),
		uintptr(bAlpha),
		uintptr(dwFlags),
		0, 0)
	return ret != 0
}
