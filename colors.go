package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	fmtColor     = "\x03"
	fmtBold      = "\x02"
	fmtItalic    = "\x1d"
	fmtUnderline = "\x1f"
	fmtReverse   = "\x16" // swap background and foreground colors
	fmtReset     = "\x0f"
)

var fmtChars = [6]string{fmtColor, fmtBold, fmtItalic, fmtUnderline, fmtReverse, fmtReset}
var fmtCharsString = "\x03\x02\x1d\x1f\x16\x0f"

const (
	colorWhite     = 0
	colorBlack     = 1
	colorNavy      = 2
	colorGreen     = 3
	colorRed       = 4
	colorMaroon    = 5
	colorPurple    = 6
	colorOrange    = 7
	colorYellow    = 8
	colorLime      = 9
	colorTeal      = 10
	colorCyan      = 11
	colorBlue      = 12
	colorPink      = 13
	colorDarkGray  = 14
	colorLightGray = 15
)

var colorPalette = [16]int{
	0xffffff, //white
	0x000000, //black
	0x000080, //navy
	0x008001, //green
	0xff0000, //red
	0x800000, //maroon
	0x6a0dad, //purple
	0xff6600, //orange
	0xffff00, //yellow
	0x32cd32, //lime
	0x008080, //teal
	0x00ffff, //cyan
	0x0000ff, //blue
	0xff00ff, //pink
	0x676767, //dark gray
	0xcccccc, //light gray
}

var colorPaletteWindows = [16]int{}

func loadColorPalette() {
	for i, c := range colorPalette {
		colorPaletteWindows[i] = c&0xff<<16 | c&0xff00 | c&0xff0000>>16
	}
}

func init() {
	loadColorPalette()
}

type richtext struct {
	str       string   // stripped of all control characters
	fgColors  [][3]int // color, start, end (offset relative to str)
	bgColors  [][3]int // color, start, end (offset relative to str)
	bold      [][2]int // start,end offsets relative to str
	italic    [][2]int // start,end offsets relative to str
	underline [][2]int // start,end offsets relative to str
}

func findNumber(str string) (number, start, end int, err error) {
	s := -1
	e := -1
	for i, b := range str {
		if b >= '0' && b <= '9' {
			if s == -1 {
				s = i
			}
			e = i
		} else if s != -1 {
			break
		}
	}
	if s == -1 {
		return 0, s, e, errors.New("findNumber: no number in string")
	}
	intval, err := strconv.Atoi(str[s : e+1])
	return intval, s, e, err
}

func parseString(str string) *richtext {
	rt := &richtext{}

	for {
		i := strings.IndexAny(str, fmtCharsString)
		if i == -1 {
			break
		}
		fmtCode := string(str[i])
		str = str[:i] + str[i+1:]
		if len(str) == 0 {
			break
		}
		switch fmtCode {
		case fmtColor:
			fg, s, e, err := findNumber(str[i:])
			if err != nil || s != 0 || e > 1 {
				break
			}
			str = str[:s+i] + str[e+1+i:]

			if len(rt.fgColors) > 0 {
				rt.fgColors[len(rt.fgColors)-1][2] = i
			}

			rt.fgColors = append(rt.fgColors, [3]int{fg, i})
			if str[i] == ',' {
				str = str[:i] + str[i+1:]
				bg, s, e, err := findNumber(str[i:])
				if err != nil || s != 0 || e > 1 {
					continue
				}
				str = str[:s+i] + str[e+1+i:]
				if len(rt.bgColors) > 0 {
					rt.fgColors[len(rt.fgColors)-1][2] = i
				}
				rt.bgColors = append(rt.bgColors, [3]int{bg, i})
			}
		case fmtBold:
			if len(rt.bold) > 0 {
				rt.bold[len(rt.bold)-1][1] = i
			}
			rt.bold = append(rt.bold, [2]int{i})
		case fmtItalic:
			if len(rt.italic) > 0 {
				rt.italic[len(rt.italic)-1][1] = i
			}
			rt.italic = append(rt.italic, [2]int{i})
		case fmtUnderline:
			if len(rt.underline) > 0 {
				rt.underline[len(rt.underline)-1][1] = i
			}
			rt.underline = append(rt.underline, [2]int{i})
		case fmtReverse:

		case fmtReset:
			if rt.fgColors != nil {
				rt.fgColors[len(rt.fgColors)-1][2] = i
			}
			if rt.bgColors != nil {
				rt.bgColors[len(rt.bgColors)-1][2] = i
			}
			if rt.bold != nil {
				rt.bold[len(rt.bold)-1][1] = i
			}
			if rt.italic != nil {
				rt.italic[len(rt.italic)-1][1] = i
			}
			if rt.underline != nil {
				rt.underline[len(rt.underline)-1][1] = i
			}
		}
	}
	rt.str = str

	if rt.fgColors != nil {
		lastIdx := len(rt.fgColors) - 1
		if rt.fgColors[lastIdx][2] == 0 {
			rt.fgColors[lastIdx][2] = len(str)
		}
	}
	if rt.bgColors != nil {
		lastIdx := len(rt.bgColors) - 1
		if rt.bgColors[lastIdx][2] == 0 {
			rt.bgColors[lastIdx][2] = len(str)
		}
	}
	if rt.bold != nil {
		lastIdx := len(rt.bold) - 1
		if rt.bold[lastIdx][1] == 0 {
			rt.bold[lastIdx][1] = len(str)
		}
	}
	if rt.italic != nil {
		lastIdx := len(rt.fgColors) - 1
		if rt.italic[lastIdx][1] == 0 {
			rt.italic[lastIdx][1] = len(str)
		}
	}
	if rt.underline != nil {
		lastIdx := len(rt.fgColors) - 1
		if rt.underline[lastIdx][1] == 0 {
			rt.underline[lastIdx][1] = len(str)
		}
	}
	return rt
}

func colorString(str string, col ...int) string {
	if len(col) > 1 {
		return fmt.Sprintf("\x03%d,%d%s\x0f", col[0], col[1], str)
	} else {
		return fmt.Sprintf("\x03%d%s\x0f", col[0], str)
	}
}
