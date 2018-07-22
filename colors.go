package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

func italic(text string) string    { return string(fmtItalic) + text + string(fmtItalic) }
func bold(text string) string      { return string(fmtBold) + text + string(fmtBold) }
func underline(text string) string { return string(fmtUnderline) + text + string(fmtUnderline) }
func strikethrough(text string) string {
	return string(fmtStrikethrough) + text + string(fmtStrikethrough)
}

const (
	White     = 0
	Black     = 1
	Navy      = 2
	Green     = 3
	Red       = 4
	Maroon    = 5
	Purple    = 6
	Orange    = 7
	Yellow    = 8
	Lime      = 9
	Teal      = 10
	Cyan      = 11
	Blue      = 12
	Pink      = 13
	DarkGray  = 14
	LightGray = 15
	DarkGrey  = 14
	LightGrey = 15 // >_>
)

func color(text string, colors ...int) string {
	str := ""
	if len(colors) > 0 {
		str = fmt.Sprintf("%02d", colors[0])
	}
	if len(colors) > 1 {
		str += fmt.Sprintf(",%02d", colors[1])
	}
	return string(fmtColor) + str + text + string(fmtReset)
}

const (
	fmtColor         = '\x03'
	fmtBold          = '\x02'
	fmtItalic        = '\x1d'
	fmtStrikethrough = '\x1e'
	fmtUnderline     = '\x1f'
	fmtReverse       = '\x16' // TODO(tso): swap background and foreground colors
	fmtReset         = '\x0f'
)

var fmtCharsString = "\x03\x02\x1d\x1e\x1f\x16\x0f"
var fmtCharsRunes = []rune{'\x03', '\x03', '\x02', '\x1d', '\x1e', '\x1f', '\x16', '\x0f'}

var defaultColorPalette = [99]int{
	// 0 - 15: standard irc colors, see constants above for indices => color
	0xffffff, 0x000000, 0x000080, 0x008001, 0xff0000, 0x800000, 0x6a0dad, 0xff6600,
	0xffff00, 0x32cd32, 0x008080, 0x00ffff, 0x0000ff, 0xff00ff, 0x676767, 0xcccccc,
	// 16-98: http://modern.ircdocs.horse/formatting.html#colors-16-98
	0x470000, 0x472100, 0x474700, 0x324700, 0x004700, 0x00472c, 0x004747, 0x002747,
	0x000047, 0x2e0047, 0x470047, 0x47002a, 0x740000, 0x743a00, 0x747400, 0x517400,
	0x007400, 0x007449, 0x007474, 0x004074, 0x000074, 0x4b0074, 0x740074, 0x740045,
	0xb50000, 0xb56300, 0xb5b500, 0x7db500, 0x00b500, 0x00b571, 0x00b5b5, 0x0063b5,
	0x0000b5, 0x7500b5, 0xb500b5, 0xb5006b, 0xff0000, 0xff8c00, 0xffff00, 0xb2ff00,
	0x00ff00, 0x00ffa0, 0x00ffff, 0x008cff, 0x0000ff, 0xa500ff, 0xff00ff, 0xff0098,
	0xff5959, 0xffb459, 0xffff71, 0xcfff60, 0x6fff6f, 0x65ffc9, 0x6dffff, 0x59b4ff,
	0x5959ff, 0xc459ff, 0xff66ff, 0xff59bc, 0xff9c9c, 0xffd39c, 0xffff9c, 0xe2ff9c,
	0x9cff9c, 0x9cffdb, 0x9cffff, 0x9cd3ff, 0x9c9cff, 0xdc9cff, 0xff9cff, 0xff94d3,
	0x000000, 0x131313, 0x282828, 0x363636, 0x4d4d4d, 0x656565, 0x818181, 0x9f9f9f,
	0xbcbcbc, 0xe2e2e2, 0xffffff,
	// NOTE(tso): 99 is same as \x0f
}

var globalBackgroundColor = 0xffffff
var globalForegroundColor = 0x000000

func loadPaletteFromFile(filename string) ([]int, error) {
	f, err := os.Open(THEMES_DIR + filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	csv, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	values := [][]byte{}
	tmp := []byte{}
	for _, b := range csv {
		if b == ',' {
			values = append(values, tmp)
			tmp = []byte{}
			continue
		}
		tmp = append(tmp, b)
	}
	values = append(values, tmp)
	palette := []int{}
	for _, v := range values {
		c, err := strconv.Atoi(string(v))
		if err != nil {
			return nil, err
		}
		palette = append(palette, c)
	}
	return palette, nil
}

var colorPalette = [99]int{}
var colorPaletteWindows = [99]int{}

func loadColorPalette(palette []int) {
	for i, c := range palette {
		colorPalette[i] = c
		colorPaletteWindows[i] = c&0xff<<16 | c&0xff00 | c&0xff0000>>16
	}
}

func init() {
	loadColorPalette(defaultColorPalette[:])
}

func matchRune(r rune, any []rune) bool {
	for _, a := range any {
		if r == a {
			return true
		}
	}
	return false
}

// NOTE(tso): maxlen is 0-indexed, 0 == 1 digit, 1 == 2 digits, ...
func findNumber(runes []rune, maxlen int) (number, start, end int, err error) {
	s := -1
	e := -1

	for i, r := range runes {
		if r >= '0' && r <= '9' {
			if s == -1 {
				s = i
			}
			e = i
			if e-s >= maxlen {
				break
			}
		} else if s != -1 {
			break
		}
	}
	if s == -1 {
		return 0, s, e, errors.New("findNumber: no number in string")
	}
	runes = runes[s : e+1]
	intval, err := strconv.Atoi(string(runes))
	return intval, s, e, err
}

func clearLast(styles [][]int, TextEffectCode int, index int) ([][]int, bool) {
	match := false
	for i, style := range styles {
		if (style[0] == TextEffectCode || TextEffectCode == TextEffectReset) && style[2] == 0 {
			styles[i][2] = index
			match = true
		}
	}
	return styles, match
}

func parseString(str string) (text string, styles [][]int) {
	styles = [][]int{}

	if strings.IndexAny(str, fmtCharsString) == -1 {
		return str, styles
	}

	runes := []rune(str)

	i := 0
	for {
		if i > len(runes)-1 {
			break
		}
		r := runes[i]
		if !matchRune(r, fmtCharsRunes) {
			i++
			continue
		}

		fmtCode := r

		runes = append(runes[:i], runes[i+1:]...)

		if len(runes) == 0 {
			break
		}
		if fmtCode != fmtReset {
			if fmtCode == fmtColor {
				fg, s, e, err := findNumber(runes[i:], 1)
				if err != nil || s != 0 || e > 1 {
					continue
				}

				runes = append(runes[:s+i], runes[e+1+i:]...)
				if fg == 99 {
					styles, _ = clearLast(styles, TextEffectReset, i)
					continue
				}

				styles, _ = clearLast(styles, TextEffectForegroundColor, i)
				styles = append(styles, []int{TextEffectForegroundColor, i, 0, colorPaletteWindows[fg]})

				if runes[i] == ',' {
					runes = append(runes[:i], runes[i+1:]...)

					bg, s, e, err := findNumber(runes[i:], 1)
					if err != nil || s != 0 || e > 1 {
						continue
					}

					runes = append(runes[:s+i], runes[e+1+i:]...)

					styles, _ = clearLast(styles, TextEffectBackgroundColor, i)
					styles = append(styles, []int{TextEffectBackgroundColor, i, 0, colorPaletteWindows[bg]})
				}
			} else {
				var m bool
				styles, m = clearLast(styles, int(fmtCode), i)
				if !m {
					styles = append(styles, []int{int(fmtCode), i, 0})
				}
			}
		} else {
			styles, _ = clearLast(styles, TextEffectReset, i)
		}
	}

	styles, _ = clearLast(styles, TextEffectReset, len(runes))

	return string(runes), styles
}

func stripFmtChars(str string) string {
	if strings.IndexAny(str, fmtCharsString) == -1 {
		return str
	}

	runes := []rune(str)

	i := 0
	for {
		if i > len(runes)-1 {
			break
		}
		r := runes[i]
		if !matchRune(r, fmtCharsRunes) {
			i++
			continue
		}

		fmtCode := r

		runes = append(runes[:i], runes[i+1:]...)

		if len(runes) == 0 {
			break
		}
		if fmtCode == fmtColor {
			_, s, e, err := findNumber(runes[i:], 1)
			if err != nil || s != 0 || e > 1 {
				continue
			}

			runes = append(runes[:s+i], runes[e+1+i:]...)

			if runes[i] == ',' {
				runes = append(runes[:i], runes[i+1:]...)

				_, s, e, err := findNumber(runes[i:], 1)
				if err != nil || s != 0 || e > 1 {
					continue
				}

				runes = append(runes[:s+i], runes[e+1+i:]...)

			}
		}
	}

	return string(runes)
}

func colorVisible(fg, bg int) bool {
	return contrast(fg, bg) >= 4.5
}

func unpackColorFloat(color int) (r, g, b float64) {
	return float64((color >> 16) & 0xff),
		float64((color >> 8) & 0xff),
		float64(color & 0xff)
}

// returns the contrast ratio of 24-bit int colors fg and bg (foreground and background)
func contrast(fg, bg int) float64 {
	lum1 := luminance(unpackColorFloat(fg))
	lum2 := luminance(unpackColorFloat(bg))
	return math.Max(lum1, lum2) / math.Min(lum1, lum2)
}

// http://www.w3.org/TR/2008/REC-WCAG20-20081211/#relativeluminancedef
func luminance(red, green, blue float64) float64 {
	red /= 255.0
	if red < 0.03928 {
		red /= 12.92
	} else {
		red = math.Pow((red+0.055)/1.055, 2.4)
	}
	green /= 255.0
	if green < 0.03928 {
		green /= 12.92
	} else {
		green = math.Pow((green+0.055)/1.055, 2.4)
	}
	blue /= 255.0
	if blue < 0.03928 {
		blue /= 12.92
	} else {
		blue = math.Pow((blue+0.055)/1.055, 2.4)
	}
	return (0.2126 * red) + (0.7152 * green) + (0.0722 * blue)
}

func rgb2COLORREF(rgb int) win.COLORREF {
	return win.COLORREF(rgb&0xff<<16 | rgb&0xff00 | rgb&0xff0000>>16)
}

func rgb2RGB(rgb int) walk.Color {
	b, g, r := byte(rgb&0xff), byte((rgb>>8)&0xff), byte((rgb>>16)&0xff)
	return walk.RGB(r, g, b)
}

func rgb2Brush(rgb int) *walk.SolidColorBrush {
	brush, err := walk.NewSolidColorBrush(rgb2RGB(rgb))
	checkErr(err)
	return brush
}

func applyThemeToRichEdit(re *RichEdit) {
	re.SendMessage(win.WM_USER+67, 0, uintptr(rgb2COLORREF(globalBackgroundColor)))
	charfmt := _charformat{
		dwMask:      CFM_COLOR,
		crTextColor: uint32(rgb2COLORREF(globalForegroundColor)),
	}
	charfmt.cbSize = uint32(unsafe.Sizeof(charfmt))
	re.SendMessage(EM_SETCHARFORMAT, 0, uintptr(unsafe.Pointer(&charfmt)))
}

func applyThemeToLineEdit(le *walk.LineEdit, brush *walk.SolidColorBrush, rgb walk.Color) {
	le.SetBackground(brush)
	le.SetTextColor(rgb)
}

func applyThemeToTabPage(tp *walk.TabPage, brush *walk.SolidColorBrush) {
	tp.SetBackground(brush)
	win.SetTextColor(win.GetDC(tp.Handle()), rgb2COLORREF(globalForegroundColor))
}

func applyThemeToTab(t tab, brush *walk.SolidColorBrush, rgb walk.Color) {
	switch t.(type) {
	case *tabServer:
		t := t.(*tabServer)
		applyThemeToTabPage(t.tabPage, brush)
		applyThemeToRichEdit(t.textBuffer)
		applyThemeToLineEdit(&t.textInput.LineEdit, brush, rgb)
	case *tabChannel:
		t := t.(*tabChannel)
		applyThemeToTabPage(t.tabPage, brush)
		applyThemeToRichEdit(t.textBuffer)
		applyThemeToLineEdit(&t.textInput.LineEdit, brush, rgb)
		applyThemeToLineEdit(t.topicInput, brush, rgb)
		t.nickListBox.SetBackground(brush)
	case *tabPrivmsg:
		t := t.(*tabPrivmsg)
		applyThemeToTabPage(t.tabPage, brush)
		applyThemeToRichEdit(t.textBuffer)
		applyThemeToLineEdit(&t.textInput.LineEdit, brush, rgb)
	default:
		log.Printf("type %T does not support theming yet!!", t)
	}
}

func applyTheme(filename string) {
	userTheme, err := loadPaletteFromFile(filename)
	checkErr(err)
	loadColorPalette(userTheme[:16])
	bg := userTheme[16]
	fg := userTheme[17]
	globalBackgroundColor = bg
	globalForegroundColor = fg

	brush := rgb2Brush(bg)
	rgb := rgb2RGB(globalForegroundColor)

	mw.SetBackground(brush)
	tabWidget.SetBackground(brush)
	mw.StatusBar().SetBackground(brush)

	for _, t := range clientState.tabs {
		applyThemeToTab(t, brush, rgb)
	}
}
