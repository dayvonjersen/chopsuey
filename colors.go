package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FIXME(tso): these are supposed to work like toggles not reset formatting at end
func italic(text string) string    { return fmtItalic + text + fmtReset }
func bold(text string) string      { return fmtBold + text + fmtReset }
func underline(text string) string { return fmtUnderline + text + fmtReset }

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
	return fmtColor + str + text + fmtReset
}

const (
	fmtColor     = "\x03"
	fmtBold      = "\x02"
	fmtItalic    = "\x1d"
	fmtUnderline = "\x1f"
	fmtReverse   = "\x16" // swap background and foreground colors
	fmtReset     = "\x0f"

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

func findNumber(str string, maxlen int) (number, start, end int, err error) {
	s := -1
	e := -1
	for i, b := range str {
		if b >= '0' && b <= '9' {
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
	intval, err := strconv.Atoi(str[s : e+1])
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
				fg, s, e, err := findNumber(str[i:], 1)
				if err != nil || s != 0 || e > 1 {
					break
				}
				str = str[:s+i] + str[e+1+i:]
				if fg > 15 { // FIXME(tso): eating \x03xx where x > 15 without displaying any color or resetting previous color is WRONG
					break
				}

				styles, _ = clearLast(styles, TextEffectForegroundColor, i)
				styles = append(styles, []int{TextEffectForegroundColor, i, 0, colorPaletteWindows[fg]})

				if str[i] == ',' {
					str = str[:i] + str[i+1:]
					bg, s, e, err := findNumber(str[i:], 1)
					if err != nil || s != 0 || e > 1 {
						continue
					}

					str = str[:s+i] + str[e+1+i:]
					if bg > 15 { // FIXME(tso): eating \x03xx where x > 15 without displaying any color or resetting previous color is WRONG
						break
					}

					styles, _ = clearLast(styles, TextEffectBackgroundColor, i)
					styles = append(styles, []int{TextEffectBackgroundColor, i, 0, colorPaletteWindows[bg]})
				}
			} else {
				var match bool
				styles, match = clearLast(styles, int(rune(fmtCode[0])), i)
				if !match {
					styles = append(styles, []int{int(rune(fmtCode[0])), i, 0})
				}
			}
		} else {
			styles, _ = clearLast(styles, TextEffectReset, i)
		}
	}
	styles, _ = clearLast(styles, TextEffectReset, len(str))
	return str, styles
}
