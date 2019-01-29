/*
Copyright © 2018 tso <tso@teknik.io>

This work is free. You can redistribute it and/or modify it under the
terms of the Do What The Fuck You Want To Public License, Version 2,
as published by Sam Hocevar. See the LICENSE.txt file for more details.
*/
package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

func init() {
	win.MustLoadLibrary("Msftedit.dll")
}

// NOTE(tso): These constants and struct definitions are taken from richedit.h
//
//            The version of richedit.h that comes with 32-bit mingw/tdm-gcc is outdated and incomplete.
//            Refer to the one distributed with 64-bit versions of mingw, available online here:
//            https://github.com/Alexpux/mingw-w64/blob/master/mingw-w64-headers/include/richedit.h
//
//            I've reduced this list to just the constants this package actually uses
//            and might potentially use in the future.
//
//            I might change them to lowercase camelCase so they're unexported in future
// -tso 7/11/2018 5:30:45 PM
const (
	CFM_BOLD        = 1
	CFM_ITALIC      = 2
	CFM_UNDERLINE   = 4
	CFM_STRIKEOUT   = 8
	CFM_PROTECTED   = 16
	CFM_LINK        = 32
	CFM_SIZE        = 0x80000000
	CFM_COLOR       = 0x40000000
	CFM_FACE        = 0x20000000
	CFM_OFFSET      = 0x10000000
	CFM_CHARSET     = 0x08000000
	CFM_SUBSCRIPT   = 0x00010000
	CFM_SUPERSCRIPT = 0x00030000
	CFE_AUTOCOLOR   = 0x40000000
	CFM_EFFECTS     = (CFM_BOLD | CFM_ITALIC | CFM_UNDERLINE | CFM_COLOR | CFM_STRIKEOUT | CFM_PROTECTED | CFM_LINK)

	CFM_BACKCOLOR = 0x04000000

	EM_SETCHARFORMAT    = (win.WM_USER + 68)
	EM_SETEVENTMASK     = (win.WM_USER + 69)
	EM_GETTEXTRANGE     = (win.WM_USER + 75)
	EM_AUTOURLDETECT    = (win.WM_USER + 91)
	EM_GETAUTOURLDETECT = (win.WM_USER + 92)

	EM_SETEDITSTYLE = (win.WM_USER + 204)
	EM_GETEDITSTYLE = (win.WM_USER + 205)
	EM_GETSCROLLPOS = (win.WM_USER + 221)
	EM_SETSCROLLPOS = (win.WM_USER + 222)

	EN_LINK = 1803

	ENM_NONE            = 0
	ENM_DRAGDROPDONE    = 16
	ENM_DROPFILES       = 1048576
	ENM_KEYEVENTS       = 65536
	ENM_LINK            = 67108864
	ENM_MOUSEEVENTS     = 131072
	ENM_OBJECTPOSITIONS = 33554432

	SES_EXTENDBACKCOLOR = 4
	SES_CTFALLOWEMBED   = 0x00200000
)

type _charformat struct {
	cbSize          uint32
	dwMask          uint32
	dwEffects       uint32
	yHeight         int32
	yOffset         int32
	crTextColor     uint32
	bCharSet        byte
	bPitchAndFamily byte
	szFaceName      [32]byte
}

type _charformat2 struct {
	cbSize          uint32
	dwMask          uint32
	dwEffects       uint32
	yHeight         int32
	yOffset         int32
	crTextColor     uint32
	bCharSet        byte
	bPitchAndFamily byte
	szFaceName      [32]byte
	wWeight         uint16
	sSpacing        int16
	crBackColor     uint32
	lcid            uint32
	dwReserved      uint32
	sStyle          int16
	wKerning        uint16
	bUnderlineType  byte
	bAnimation      byte
	bRevAuthor      byte
}

type _nmhdr struct {
	hwndFrom uintptr
	idFrom   uintptr
	code     uint32
}

type _chrg struct {
	cpMin, cpMax int32
}

type _enlink struct {
	nmhdr  _nmhdr
	msg    int32
	wParam uintptr
	lParam [4]byte // HACK(tso): this is actually a uintptr
	chrg   _chrg
}

type _textrange struct {
	chrg _chrg
	text []uint16
}

type RichEdit struct {
	walk.WidgetBase
	linecount int
}

//
// common interface copied from walk.TextEdit
//

//
func (re *RichEdit) LayoutFlags() walk.LayoutFlags {
	return walk.GrowableHorz | walk.GrowableVert | walk.GreedyHorz | walk.GreedyVert
}

func (re *RichEdit) MinSizeHint() walk.Size {
	return walk.Size{20, 12}
}

func (re *RichEdit) SizeHint() walk.Size {
	return walk.Size{400, 100}
}

func (re *RichEdit) TextLength() int {
	return int(re.SendMessage(win.WM_GETTEXTLENGTH, 0, 0))
}

func (re *RichEdit) TextSelection() (start, end int) {
	re.SendMessage(win.EM_GETSEL, uintptr(unsafe.Pointer(&start)), uintptr(unsafe.Pointer(&end)))
	return start, end
}

func (re *RichEdit) SetTextSelection(start, end int) {
	re.SendMessage(win.EM_SETSEL, uintptr(start), uintptr(end))
}

func (re *RichEdit) ReplaceSelectedText(text string, canUndo bool) {
	re.SendMessage(win.EM_REPLACESEL,
		uintptr(win.BoolToBOOL(canUndo)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

func (re *RichEdit) SetText(text string) {
	re.SendMessage(win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

func (re *RichEdit) SetReadOnly(readOnly bool) error {
	if 0 == re.SendMessage(win.EM_SETREADONLY, uintptr(win.BoolToBOOL(readOnly)), 0) {
		return fmt.Errorf("SendMessage(EM_SETREADONLY) failed for some reason")
	}

	return nil
}

//
// custom stuff for RichEdit
//

// NOTE(tso): it might make sense to expose these in addition to/instead of
//            the helper functions ColorText, BoldText, ...
//            because you can set multiple styles in one EM_SETCHARFORMAT
func (re *RichEdit) setCharFormat(charfmt _charformat, start, end int) {
	charfmt.cbSize = uint32(unsafe.Sizeof(charfmt))
	s, e := re.TextSelection()
	re.SetTextSelection(start, end)
	re.SendMessage(EM_SETCHARFORMAT, 1, uintptr(unsafe.Pointer(&charfmt)))
	re.SetTextSelection(s, e)
}

func (re *RichEdit) setCharFormat2(charfmt2 _charformat2, start, end int) {
	charfmt2.cbSize = uint32(unsafe.Sizeof(charfmt2))
	s, e := re.TextSelection()
	re.SetTextSelection(start, end)
	re.SendMessage(EM_SETCHARFORMAT, 1, uintptr(unsafe.Pointer(&charfmt2)))
	re.SetTextSelection(s, e)
}

func (re *RichEdit) ColorText(colorInBBGGRRByteOrder uint32, start, end int) {
	charfmt := _charformat{
		dwMask:      CFM_COLOR,
		crTextColor: colorInBBGGRRByteOrder,
	}
	re.setCharFormat(charfmt, start, end)
}

func (re *RichEdit) BackgroundColorText(colorInBBGGRRByteOrder uint32, start, end int) {
	charfmt := _charformat2{
		dwMask:      CFM_BACKCOLOR,
		crBackColor: colorInBBGGRRByteOrder,
	}
	re.setCharFormat2(charfmt, start, end)
}

func (re *RichEdit) BoldText(start, end int) {
	charfmt := _charformat{
		dwMask:    CFM_BOLD,
		dwEffects: CFM_BOLD,
	}
	re.setCharFormat(charfmt, start, end)
}

func (re *RichEdit) ItalicText(start, end int) {
	charfmt := _charformat{
		dwMask:    CFM_ITALIC,
		dwEffects: CFM_ITALIC,
	}
	re.setCharFormat(charfmt, start, end)
}

func (re *RichEdit) UnderlineText(start, end int) {
	charfmt := _charformat{
		dwMask:    CFM_UNDERLINE,
		dwEffects: CFM_UNDERLINE,
	}
	re.setCharFormat(charfmt, start, end)
}

func (re *RichEdit) StrikethroughText(start, end int) {
	charfmt := _charformat{
		dwMask:    CFM_STRIKEOUT,
		dwEffects: CFM_STRIKEOUT,
	}
	re.setCharFormat(charfmt, start, end)
}

// Removes all text effects
func (re *RichEdit) ResetText(start, end int) {
	charfmt := _charformat2{
		dwMask:    CFE_AUTOCOLOR,
		dwEffects: CFE_AUTOCOLOR,
	}
	re.setCharFormat2(charfmt, start, end)
}

// NOTE(tso): these numbers were picked based on IRC text formatting bytes
//            e.g. \x02 is "bold"
//            it's entirely arbitrary and could just as well be 0, 1, 2, 3...
// -tso 7/11/2018 5:13:12 PM
//            but it makes it consistent with how styles are returned from
//            parseString() in colors.go
// -tso 7/16/2018 9:13:51 AM
const (
	//TextEffectColor = 3      // Use TextEffectForegroundColor or TextEffectBackgroundColor
	TextEffectBold          = 2
	TextEffectItalic        = 29
	TextEffectStrikethrough = 30
	TextEffectUnderline     = 31
	TextEffectReverse       = 22
	TextEffectReset         = 15

	TextEffectForegroundColor = 102
	TextEffectBackgroundColor = 98
	//TextEffectLink = 691337 // Unnecessary since AUTOURLDETECT is enabled
)

// optionally apply styles with one or more int slices where
//  - offset 0 is one of the TextEffect constants above
//  - offset 1 is the starting byte offset in text for the desired effect
//  - offset 2 is the ending byte offset in text for the desired effect
//  - offset 3 only is required for TextEffectForegroundColor or TextEffectBackgroundColor
//                and is a 32-bit color in GGBBRR byte-order
func (re *RichEdit) AppendText(text string, styles ...[]int) {
	s, e := re.TextSelection()
	l := re.TextLength()
	// HACK(tso): something is happening here that i don't fully understand
	//            TextLength() includes \n as chars but the call to setCharFormat()
	//            ignores \n chars????
	//            this only happened when we switched to RICHEDIT20W from RICHEDIT
	//            in order to support unicode but this has nothing to with character
	//            encoding afaict
	// -tso, 7/10/2018 3:00:03 AM
	if l == 0 {
		re.linecount = 0
		if text == "\n" { // HACK(tso): hey while we're here, counting lines...
			return
			// clarification:
			//                   (tabChatbox.Println always appends a "\n" *before* appending new text
			//                   so that the end of the buffer is always flush with the edge
			//                   but you don't want to do that for the first line in the buffer
			//                   this if statement should actually be in that file, but linecount isn't
			//                   "public" though I could just change it to Linecount or better yet LineCount
			//                   but it shouldn't even have to exist at all
			// -tso 7/14/2018 7:09:42 AM
		}
	}
	re.SetTextSelection(l, l)
	re.ReplaceSelectedText(text, false)
	for _, style := range styles {
		start, end := style[1], style[2]
		start += l - re.linecount // HACK(tso): this fixes it idk why
		end += l - re.linecount   //
		switch style[0] {
		case TextEffectForegroundColor:
			re.ColorText(uint32(style[3]), start, end)
		case TextEffectBackgroundColor:
			re.BackgroundColorText(uint32(style[3]), start, end)
		case TextEffectBold:
			re.BoldText(start, end)
		case TextEffectItalic:
			re.ItalicText(start, end)
		case TextEffectUnderline:
			re.UnderlineText(start, end)
		case TextEffectStrikethrough:
			re.StrikethroughText(start, end)
		case TextEffectReset:
			re.ResetText(start, end)
		}
	}
	re.SetTextSelection(s, e)
	re.linecount += strings.Count(text, "\n") // HACK(tso): but number of new lines in the selection we just added doesnt matter? I'm so confused...
	// XXX HACK FIXME(tso): actually, styles still aren't applied correctly
	//                      if there are *any* newlines in text we add with
	//                      this function.
	//
	//                      it's not hard for _me_ to remember either to
	//                      a) never AppendText with multiple newlines and/or
	//                      b) never use any style information with such text
	//
	//                      but it's clearly not right and it's a landmine for
	//                      anyone else who might want to use this.
	// -tso 7/14/2018 7:10:52 AM
}

func (re *RichEdit) openURL(min, max int32) {
	textRange := &_textrange{
		chrg: _chrg{min, max},
		text: make([]uint16, (max - min)),
	}
	re.SendMessage(EM_GETTEXTRANGE, 0, uintptr(unsafe.Pointer(textRange)))

	url := string(utf16.Decode(textRange.text))

	cmd := exec.Command("cmd", "/c", "start", url)
	if err := cmd.Run(); err != nil {
		log.Println("cmd /c start", url, "returned error:\n", err)
	}
}

func (re *RichEdit) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {

	// open links in default web browser
	// TODO(tso): potentially dangerous, add option to disable.
	if msg == win.WM_NOTIFY {
		nmhdr := (*_nmhdr)(unsafe.Pointer(lParam))
		if nmhdr.code == EN_LINK {
			enlink := (*_enlink)(unsafe.Pointer(lParam))
			if enlink.msg == win.WM_LBUTTONUP {
				go re.openURL(enlink.chrg.cpMin, enlink.chrg.cpMax)
				return 0
			}
		}
	}

	// disable smooth scroll
	if msg == win.WM_MOUSEWHEEL {
		delta := int(int16(win.HIWORD(uint32(wParam))))
		var direction uintptr
		if delta > 0 {
			direction = win.SB_LINEUP
		} else {
			direction = win.SB_LINEDOWN
		}
		re.SendMessage(win.WM_VSCROLL, direction, 0)
		re.SendMessage(win.WM_VSCROLL, direction, 0)
		re.SendMessage(win.WM_VSCROLL, direction, 0)
		re.SendMessage(win.WM_VSCROLL, direction, 0)
		return re.SendMessage(win.WM_VSCROLL, direction, 0)
	}
	return re.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

// walk Interface
func NewRichEdit(parent walk.Container) (*RichEdit, error) {
	re := &RichEdit{}
	err := walk.InitWidget(
		re,
		parent,
		"RICHEDIT50W",
		win.ES_MULTILINE|win.WS_VISIBLE|win.WS_CHILD|win.WS_VSCROLL,
		0,
	)
	if err != nil {
		return nil, err
	}
	re.SendMessage(EM_SETEVENTMASK, 0, uintptr(ENM_LINK)) //|ENM_MOUSEEVENTS|ENM_OBJECTPOSITIONS|ENM_KEYEVENTS))
	re.SendMessage(EM_SETEDITSTYLE, 0, uintptr(SES_CTFALLOWEMBED|SES_EXTENDBACKCOLOR))
	re.SendMessage(EM_AUTOURLDETECT, 1, 0)

	re.SetAlwaysConsumeSpace(true)
	re.SetReadOnly(true)

	return re, err
}

// walk/declarative Interface
type RichEditDecl struct {
	AssignTo      **RichEdit
	StretchFactor int
}

func (re RichEditDecl) Create(builder *declarative.Builder) error {
	w, err := NewRichEdit(builder.Parent())
	if err != nil {
		return err
	}
	if re.AssignTo != nil {
		*re.AssignTo = w
	}

	return builder.InitWidget(re, w, func() error { return nil })
}
