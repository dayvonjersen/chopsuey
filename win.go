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
	// Layered Window Attributes
	LWA_COLORKEY = 1
	LWA_ALPHA    = 2

	// Owner Draw Type (win.DRAWITEMSTRUCT.CtlType)
	ODT_MENU     = 0x1
	ODT_LISTBOX  = 0x2
	ODT_COMBOBOX = 0x3
	ODT_BUTTON   = 0x4
	ODT_STATIC   = 0x5

	// Owner Draw Action (win.DRAWITEMSTRUCT.ItemAction)
	ODA_DRAWENTIRE = 0x1
	ODA_SELECT     = 0x2
	ODA_FOCUS      = 0x4

	// Owner Draw State (win.DRAWITEMSTRUCT.ItemState)
	ODS_SELECTED     = 0x0001
	ODS_GRAYED       = 0x0002
	ODS_DISABLED     = 0x0004
	ODS_CHECKED      = 0x0008
	ODS_FOCUS        = 0x0010
	ODS_DEFAULT      = 0x0020
	ODS_COMBOBOXEDIT = 0x1000
	ODS_HOTLIGHT     = 0x0040
	ODS_INACTIVE     = 0x0080
	ODS_NOACCEL      = 0x0100
	ODS_NOFOCUSRECT  = 0x0200
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
