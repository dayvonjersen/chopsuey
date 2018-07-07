// +build ignore

package main

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// all of this is from richedit.h
// most of it will ~~probably~~ __definitely__ be left unused
// but it's here for the sake of completeness

const (
	CFM_BOLD                = 1
	CFM_ITALIC              = 2
	CFM_UNDERLINE           = 4
	CFM_STRIKEOUT           = 8
	CFM_PROTECTED           = 16
	CFM_LINK                = 32
	CFM_SIZE                = 0x80000000
	CFM_COLOR               = 0x40000000
	CFM_FACE                = 0x20000000
	CFM_OFFSET              = 0x10000000
	CFM_CHARSET             = 0x08000000
	CFM_SUBSCRIPT           = 0x00030000
	CFM_SUPERSCRIPT         = 0x00030000
	CFM_EFFECTS             = (CFM_BOLD | CFM_ITALIC | CFM_UNDERLINE | CFM_COLOR | CFM_STRIKEOUT | CFE_PROTECTED | CFM_LINK)
	CFE_BOLD                = 1
	CFE_ITALIC              = 2
	CFE_UNDERLINE           = 4
	CFE_STRIKEOUT           = 8
	CFE_PROTECTED           = 16
	CFE_AUTOCOLOR           = 0x40000000
	CFE_SUBSCRIPT           = 0x00010000
	CFE_SUPERSCRIPT         = 0x00020000
	IMF_FORCENONE           = 1
	IMF_FORCEENABLE         = 2
	IMF_FORCEDISABLE        = 4
	IMF_CLOSESTATUSWINDOW   = 8
	IMF_VERTICAL            = 32
	IMF_FORCEACTIVE         = 64
	IMF_FORCEINACTIVE       = 128
	IMF_FORCEREMEMBER       = 256
	SEL_EMPTY               = 0
	SEL_TEXT                = 1
	SEL_OBJECT              = 2
	SEL_MULTICHAR           = 4
	SEL_MULTIOBJECT         = 8
	MAX_TAB_STOPS           = 32
	PFM_ALIGNMENT           = 8
	PFM_NUMBERING           = 32
	PFM_OFFSET              = 4
	PFM_OFFSETINDENT        = 0x80000000
	PFM_RIGHTINDENT         = 2
	PFM_STARTINDENT         = 1
	PFM_TABSTOPS            = 16
	PFM_BORDER              = 2048
	PFM_LINESPACING         = 256
	PFM_NUMBERINGSTART      = 32768
	PFM_NUMBERINGSTYLE      = 8192
	PFM_NUMBERINGTAB        = 16384
	PFM_SHADING             = 4096
	PFM_SPACEAFTER          = 128
	PFM_SPACEBEFORE         = 64
	PFM_STYLE               = 1024
	PFM_DONOTHYPHEN         = 4194304
	PFM_KEEP                = 131072
	PFM_KEEPNEXT            = 262144
	PFM_NOLINENUMBER        = 1048576
	PFM_NOWIDOWCONTROL      = 2097152
	PFM_PAGEBREAKBEFORE     = 524288
	PFM_RTLPARA             = 65536
	PFM_SIDEBYSIDE          = 8388608
	PFM_TABLE               = 1073741824
	PFN_BULLET              = 1
	PFE_DONOTHYPHEN         = 64
	PFE_KEEP                = 2
	PFE_KEEPNEXT            = 4
	PFE_NOLINENUMBER        = 16
	PFE_NOWIDOWCONTROL      = 32
	PFE_PAGEBREAKBEFORE     = 8
	PFE_RTLPARA             = 1
	PFE_SIDEBYSIDE          = 128
	PFE_TABLE               = 16384
	PFA_LEFT                = 1
	PFA_RIGHT               = 2
	PFA_CENTER              = 3
	PFA_JUSTIFY             = 4
	PFA_FULL_INTERuint16    = 4
	SF_TEXT                 = 1
	SF_RTF                  = 2
	SF_RTFNOOBJS            = 3
	SF_TEXTIZED             = 4
	SF_UNICODE              = 16
	SF_USECODEPAGE          = 32
	SF_NCRFORNONASCII       = 64
	SF_RTFVAL               = 0x0700
	SFF_PWD                 = 0x0800
	SFF_KEEPDOCINFO         = 0x1000
	SFF_PERSISTVIEWSCALE    = 0x2000
	SFF_PLAINRTF            = 0x4000
	SFF_SELECTION           = 0x8000
	WB_CLASSIFY             = 3
	WB_MOVEuint16LEFT       = 4
	WB_MOVEuint16RIGHT      = 5
	WB_LEFTBREAK            = 6
	WB_RIGHTBREAK           = 7
	WB_MOVEuint16PREV       = 4
	WB_MOVEuint16NEXT       = 5
	WB_PREVBREAK            = 6
	WB_NEXTBREAK            = 7
	WBF_uint16WRAP          = 16
	WBF_uint16BREAK         = 32
	WBF_OVERFLOW            = 64
	WBF_LEVEL1              = 128
	WBF_LEVEL2              = 256
	WBF_CUSTOM              = 512
	ES_DISABLENOSCROLL      = 8192
	ES_EX_NOCALLOLEINIT     = 16777216
	ES_NOIME                = 524288
	ES_NOOLEDRAGDROP        = 8
	ES_SAVESEL              = 32768
	ES_SELECTIONBAR         = 16777216
	ES_SELFIME              = 262144
	ES_SUNKEN               = 16384
	ES_VERTICAL             = 4194304
	EM_CANPASTE             = (win.WM_USER + 50)
	EM_DISPLAYBAND          = (win.WM_USER + 51)
	EM_EXGETSEL             = (win.WM_USER + 52)
	EM_EXLIMITTEXT          = (win.WM_USER + 53)
	EM_EXLINEFROMCHAR       = (win.WM_USER + 54)
	EM_EXSETSEL             = (win.WM_USER + 55)
	EM_FINDTEXT             = (win.WM_USER + 56)
	EM_FORMATRANGE          = (win.WM_USER + 57)
	EM_GETCHARFORMAT        = (win.WM_USER + 58)
	EM_GETEVENTMASK         = (win.WM_USER + 59)
	EM_GETOLEINTERFACE      = (win.WM_USER + 60)
	EM_GETPARAFORMAT        = (win.WM_USER + 61)
	EM_GETSELTEXT           = (win.WM_USER + 62)
	EM_HIDESELECTION        = (win.WM_USER + 63)
	EM_PASTESPECIAL         = (win.WM_USER + 64)
	EM_REQUESTRESIZE        = (win.WM_USER + 65)
	EM_SELECTIONTYPE        = (win.WM_USER + 66)
	EM_SETBKGNDCOLOR        = (win.WM_USER + 67)
	EM_SETCHARFORMAT        = (win.WM_USER + 68)
	EM_SETEVENTMASK         = (win.WM_USER + 69)
	EM_SETOLECALLBACK       = (win.WM_USER + 70)
	EM_SETPARAFORMAT        = (win.WM_USER + 71)
	EM_SETTARGETDEVICE      = (win.WM_USER + 72)
	EM_STREAMIN             = (win.WM_USER + 73)
	EM_STREAMOUT            = (win.WM_USER + 74)
	EM_GETTEXTRANGE         = (win.WM_USER + 75)
	EM_FINDuint16BREAK      = (win.WM_USER + 76)
	EM_SETOPTIONS           = (win.WM_USER + 77)
	EM_GETOPTIONS           = (win.WM_USER + 78)
	EM_FINDTEXTEX           = (win.WM_USER + 79)
	EM_GETuint16BREAKPROCEX = (win.WM_USER + 80)
	EM_SETuint16BREAKPROCEX = (win.WM_USER + 81)
	/* RichEdit 2.0 messages */
	EM_SETUNDOLIMIT         = (win.WM_USER + 82)
	EM_REDO                 = (win.WM_USER + 84)
	EM_CANREDO              = (win.WM_USER + 85)
	EM_GETUNDONAME          = (win.WM_USER + 86)
	EM_GETREDONAME          = (win.WM_USER + 87)
	EM_STOPGROUPTYPING      = (win.WM_USER + 88)
	EM_SETTEXTMODE          = (win.WM_USER + 89)
	EM_GETTEXTMODE          = (win.WM_USER + 90)
	EM_AUTOURLDETECT        = (win.WM_USER + 91)
	EM_GETAUTOURLDETECT     = (win.WM_USER + 92)
	EM_SETPALETTE           = (win.WM_USER + 93)
	EM_GETTEXTEX            = (win.WM_USER + 94)
	EM_GETTEXTLENGTHEX      = (win.WM_USER + 95)
	EM_SHOWSCROLLBAR        = (win.WM_USER + 96)
	EM_SETTEXTEX            = (win.WM_USER + 97)
	EM_SETPUNCTUATION       = (win.WM_USER + 100)
	EM_GETPUNCTUATION       = (win.WM_USER + 101)
	EM_SETuint16WRAPMODE    = (win.WM_USER + 102)
	EM_GETuint16WRAPMODE    = (win.WM_USER + 103)
	EM_SETIMECOLOR          = (win.WM_USER + 104)
	EM_GETIMECOLOR          = (win.WM_USER + 105)
	EM_SETIMEOPTIONS        = (win.WM_USER + 106)
	EM_GETIMEOPTIONS        = (win.WM_USER + 107)
	EM_SETLANGOPTIONS       = (win.WM_USER + 120)
	EM_GETLANGOPTIONS       = (win.WM_USER + 121)
	EM_GETIMECOMPMODE       = (win.WM_USER + 122)
	EM_FINDTEXTW            = (win.WM_USER + 123)
	EM_FINDTEXTEXW          = (win.WM_USER + 124)
	EM_RECONVERSION         = (win.WM_USER + 125)
	EM_SETBIDIOPTIONS       = (win.WM_USER + 200)
	EM_GETBIDIOPTIONS       = (win.WM_USER + 201)
	EM_SETTYPOGRAPHYOPTIONS = (win.WM_USER + 202)
	EM_GETTYPOGRAPHYOPTIONS = (win.WM_USER + 203)
	EM_SETEDITSTYLE         = (win.WM_USER + 204)
	EM_GETEDITSTYLE         = (win.WM_USER + 205)
	EM_GETSCROLLPOS         = (win.WM_USER + 221)
	EM_SETSCROLLPOS         = (win.WM_USER + 222)
	EM_SETFONTSIZE          = (win.WM_USER + 223)
	EM_GETZOOM              = (win.WM_USER + 224)
	EM_SETZOOM              = (win.WM_USER + 225)

	EN_CORRECTTEXT          = 1797
	EN_DROPFILES            = 1795
	EN_IMECHANGE            = 1799
	EN_LINK                 = 1803
	EN_MSGFILTER            = 1792
	EN_OLEOPFAILED          = 1801
	EN_PROTECTED            = 1796
	EN_REQUESTRESIZE        = 1793
	EN_SAVECLIPBOARD        = 1800
	EN_SELCHANGE            = 1794
	EN_STOPNOUNDO           = 1798
	ENM_NONE                = 0
	ENM_CHANGE              = 1
	ENM_CORRECTTEXT         = 4194304
	ENM_DRAGDROPDONE        = 16
	ENM_DROPFILES           = 1048576
	ENM_IMECHANGE           = 8388608
	ENM_KEYEVENTS           = 65536
	ENM_LANGCHANGE          = 16777216
	ENM_LINK                = 67108864
	ENM_MOUSEEVENTS         = 131072
	ENM_OBJECTPOSITIONS     = 33554432
	ENM_PROTECTED           = 2097152
	ENM_REQUESTRESIZE       = 262144
	ENM_SCROLL              = 4
	ENM_SCROLLEVENTS        = 8
	ENM_SELCHANGE           = 524288
	ENM_UPDATE              = 2
	ECO_AUTOuint16SELECTION = 1
	ECO_AUTOVSCROLL         = 64
	ECO_AUTOHSCROLL         = 128
	ECO_NOHIDESEL           = 256
	ECO_READONLY            = 2048
	ECO_WANTRETURN          = 4096
	ECO_SAVESEL             = 0x8000
	ECO_SELECTIONBAR        = 0x1000000
	ECO_VERTICAL            = 0x400000
	ECOOP_SET               = 1
	ECOOP_OR                = 2
	ECOOP_AND               = 3
	ECOOP_XOR               = 4
	SCF_DEFAULT             = 0
	SCF_SELECTION           = 1
	SCF_uint16              = 2
	SCF_ALL                 = 4
	SCF_USEUIRULES          = 8
	TM_PLAINTEXT            = 1
	TM_RICHTEXT             = 2
	TM_SINGLELEVELUNDO      = 4
	TM_MULTILEVELUNDO       = 8
	TM_SINGLECODEPAGE       = 16
	TM_MULTICODEPAGE        = 32
	yHeightCharPtsMost      = 1638
	lDefaultTab             = 720

	/* GETTEXTEX flags */
	GT_DEFAULT   = 0
	GT_USECRLF   = 1
	GT_SELECTION = 2
	/* SETTEXTEX flags */
	ST_DEFAULT   = 0
	ST_KEEPUNDO  = 1
	ST_SELECTION = 2
	/* Defines for EM_SETTYPOGRAPHYOPTIONS */
	TO_ADVANCEDTYPOGRAPHY = 1
	TO_SIMPLELINEBREAK    = 2
	/* Defines for GETTEXTLENGTHEX */
	GTL_DEFAULT  = 0
	GTL_USECRLF  = 1
	GTL_PRECISE  = 2
	GTL_CLOSE    = 4
	GTL_NUMCHARS = 8
	GTL_NUMbyteS = 16
)

// ALL THOSE CONSTANTS AND THIS WASN'T DEFINED
const CFM_BACKCOLOR = 0x04000000

// tbf it's in 64-bit mingw header files

type charformat struct {
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

type charformat2 struct {
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

func RGBToColorRef(r, g, b int) uint32 {
	return uint32(r&0xff | (g&0xff)<<8 | (b&0xff)<<16)
}

func ColorRefToRGB(cr uint32) (r, g, b int) {
	{
		cr := int(cr)
		r = cr & 0xff
		g = cr >> 8 & 0xff
		b = cr >> 16 & 0xff
	}
	return r, g, b
}

func init() {
	win.MustLoadLibrary("Riched32.dll")
}

type RichEdit struct {
	walk.WidgetBase
}

func (re *RichEdit) SetCharFormat(charfmt charformat, start, end int) {
	charfmt.cbSize = uint32(unsafe.Sizeof(charfmt))
	s, e := re.TextSelection()
	re.SetTextSelection(start, end)
	re.SendMessage(EM_SETCHARFORMAT, 1, uintptr(unsafe.Pointer(&charfmt)))
	re.SetTextSelection(s, e)
}

func (re *RichEdit) SetCharFormat2(charfmt charformat2, start, end int) {
	charfmt.cbSize = uint32(unsafe.Sizeof(charfmt))
	s, e := re.TextSelection()
	re.SetTextSelection(start, end)
	re.SendMessage(EM_SETCHARFORMAT, 1, uintptr(unsafe.Pointer(&charfmt)))
	re.SetTextSelection(s, e)
}

func (re *RichEdit) Color(r, g, b, start, end int) {
	charfmt := charformat{
		dwMask:      CFM_COLOR,
		crTextColor: RGBToColorRef(r, g, b),
	}
	re.SetCharFormat(charfmt, start, end)
}

func (re *RichEdit) BackgroundColor(r, g, b, start, end int) {
	charfmt := charformat2{
		dwMask:      CFM_BACKCOLOR,
		crBackColor: RGBToColorRef(r, g, b),
	}
	re.SetCharFormat2(charfmt, start, end)
}

func (re *RichEdit) Bold(start, end int) {
	charfmt := charformat{
		dwMask:    CFM_BOLD,
		dwEffects: CFM_BOLD,
	}
	re.SetCharFormat(charfmt, start, end)
}

func (re *RichEdit) Italic(start, end int) {
	charfmt := charformat{
		dwMask:    CFM_ITALIC,
		dwEffects: CFM_ITALIC,
	}
	re.SetCharFormat(charfmt, start, end)
}

func (re *RichEdit) Underline(start, end int) {
	charfmt := charformat{
		dwMask:    CFM_UNDERLINE,
		dwEffects: CFM_UNDERLINE,
	}
	re.SetCharFormat(charfmt, start, end)
}

func (re *RichEdit) LayoutFlags() walk.LayoutFlags {
	return walk.ShrinkableHorz | walk.ShrinkableVert | walk.GrowableHorz | walk.GrowableVert | walk.GreedyHorz | walk.GreedyVert
}

func (re *RichEdit) MinSizeHint() walk.Size {
	return walk.Size{20, 12}
}

func (re *RichEdit) SizeHint() walk.Size {
	return walk.Size{100, 100}
}

func (re *RichEdit) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	return re.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

func (re *RichEdit) TextLength() int {
	return int(re.SendMessage(win.WM_GETTEXTLENGTH, 0, 0))
}

func (re *RichEdit) TextSelection() (start, end int) {
	re.SendMessage(win.EM_GETSEL, uintptr(unsafe.Pointer(&start)), uintptr(unsafe.Pointer(&end)))
	return
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

func (re *RichEdit) AppendText(text string) {
	s, e := re.TextSelection()
	l := re.TextLength()
	re.SetTextSelection(l, l)
	re.ReplaceSelectedText(text, false)
	re.SetTextSelection(s, e)
}

func (re *RichEdit) SetReadOnly(readOnly bool) error {
	if 0 == re.SendMessage(win.EM_SETREADONLY, uintptr(win.BoolToBOOL(readOnly)), 0) {
		return fmt.Errorf("SendMessage(EM_SETREADONLY) failed for some reason")
	}

	return nil
}

func NewRichEdit(parent walk.Container) (*RichEdit, error) {
	re := &RichEdit{}
	err := walk.InitWidget(
		re,
		parent,
		"RICHEDIT",
		win.ES_MULTILINE|win.WS_VISIBLE|win.WS_CHILD|win.WS_BORDER|win.WS_VSCROLL,
		win.WS_EX_CLIENTEDGE,
	)
	if err != nil {
		return nil, err
	}
	re.SetAlwaysConsumeSpace(true)
	return re, err
}

/*
if __name__ == "__main__":
*/
func main() {
	var mw *walk.MainWindow

	MainWindow{
		AssignTo: &mw,
		Title:    "Rich Edit Test",
		MinSize:  Size{600, 400},
		Layout:   VBox{},
	}.Create()

	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)

	re, err := NewRichEdit(mw)
	checkErr(err)
	// checkErr(re.SetReadOnly(true))

	str := "this is a \x0313te\x0fst http://www.google.com"
	rt := parseString(str)
	re.SetText(rt.str)
	//	r, g, b := ColorRefToRGB()

	color := colorPaletteWindows[rt.fgColors[0][0]]
	r := color >> 16 & 0xff
	g := color >> 8 & 0xff
	b := color & 0xff
	re.Color(r, g, b, rt.fgColors[0][1], rt.fgColors[0][2])
	//	re.Color(0x99, 0xcc, 0xff, 0, 4)
	//	re.BackgroundColor(0x66, 0xee, 0x77, 0, 9)
	//	re.Bold(10, 14)
	//	re.SetCharFormat(charformat{
	//		dwMask:    32,
	//		dwEffects: 32,
	//	}, 15, 36)
	go func() {
		<-time.After(time.Second)
		mw.WindowBase.Synchronize(func() {
			l := re.TextLength()
			re.SetTextSelection(l, l)
		})
	}()

	mw.Run()
}

func checkErr(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
