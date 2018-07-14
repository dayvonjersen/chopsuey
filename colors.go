package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FIXME(tso): these are supposed to work like toggles
//             e.g. " \x1f italics ON \x1f italics OFF "
//             we shouldn't have to hard reset formatting at end
// -tso 7/14/2018 2:19:06 AM
func italic(text string) string    { return string(fmtItalic) + text + string(fmtReset) }
func bold(text string) string      { return string(fmtBold) + text + string(fmtReset) }
func underline(text string) string { return string(fmtUnderline) + text + string(fmtReset) }

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
	fmtColor     = '\x03'
	fmtBold      = '\x02'
	fmtItalic    = '\x1d'
	fmtUnderline = '\x1f'
	fmtReverse   = '\x16' // TODO(tso): swap background and foreground colors
	fmtReset     = '\x0f'
)

var fmtCharsString = "\x03\x02\x1d\x1f\x16\x0f"
var fmtCharsRunes = []rune{'\x03', '\x03', '\x02', '\x1d', '\x1f', '\x16', '\x0f'}

var colorPalette = [16]int{ // TODO(tso): load palettes from files and more than 16 colors
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

func matchRune(r rune, any []rune) bool {
	for _, a := range any {
		if r == a {
			return true
		}
	}
	return false
}

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
					break
				}

				runes = append(runes[:s+i], runes[e+1+i:]...)
				if fg > 15 { // FIXME(tso): eating \x03xx where x > 15 without displaying any color or resetting previous color is WRONG
					break
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
					if bg > 15 { // FIXME(tso): eating \x03xx where x > 15 without displaying any color or resetting previous color is WRONG
						break
					}

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
