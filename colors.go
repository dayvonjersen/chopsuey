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

	//styleColor           = 3
	styleBold            = 2
	styleItalic          = 29
	styleUnderline       = 31
	styleReverse         = 22
	styleReset           = 15
	styleForegroundColor = 102
	styleBackgroundColor = 98

	fmtWhite     = "\x030"
	fmtBlack     = "\x031"
	fmtNavy      = "\x032"
	fmtGreen     = "\x033"
	fmtRed       = "\x034"
	fmtMaroon    = "\x035"
	fmtPurple    = "\x036"
	fmtOrange    = "\x037"
	fmtYellow    = "\x038"
	fmtLime      = "\x039"
	fmtTeal      = "\x0310"
	fmtCyan      = "\x0311"
	fmtBlue      = "\x0312"
	fmtPink      = "\x0313"
	fmtDarkGray  = "\x0314"
	fmtLightGray = "\x0315"
)

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
	str    string  // stripped of all control characters
	styles [][]int // style type, start offset, end offset, color value
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

func clearLast(styles [][]int, styleCode int, index int) [][]int {
	for i, style := range styles {
		if (style[0] == styleCode || styleCode == styleReset) && style[2] == 0 {
			styles[i][2] = index
		}
	}
	return styles
}

func parseString(str string) *richtext {
	styles := [][]int{}
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
		if fmtCode != fmtReset {
			if fmtCode == fmtColor {
				fg, s, e, err := findNumber(str[i:])
				if err != nil || s != 0 || e > 1 {
					break
				}
				str = str[:s+i] + str[e+1+i:]

				styles = clearLast(styles, styleForegroundColor, i)
				styles = append(styles, []int{styleForegroundColor, i, 0, colorPaletteWindows[fg]})

				if str[i] == ',' {
					str = str[:i] + str[i+1:]
					bg, s, e, err := findNumber(str[i:])
					if err != nil || s != 0 || e > 1 {
						continue
					}

					str = str[:s+i] + str[e+1+i:]

					styles = clearLast(styles, styleBackgroundColor, i)
					styles = append(styles, []int{styleBackgroundColor, i, 0, colorPaletteWindows[bg]})
				}
			} else {
				styles = clearLast(styles, int(rune(fmtCode[0])), i)
				styles = append(styles, []int{int(rune(fmtCode[0])), i, 0})
			}
		} else {
			styles = clearLast(styles, styleReset, i)
		}
	}
	styles = clearLast(styles, styleReset, len(str))
	return &richtext{str, styles}
}

func colorString(str string, col ...int) string {
	if len(col) > 1 {
		return fmt.Sprintf("\x03%d,%d%s\x0f", col[0], col[1], str)
	} else {
		return fmt.Sprintf("\x03%d%s\x0f", col[0], str)
	}
}
