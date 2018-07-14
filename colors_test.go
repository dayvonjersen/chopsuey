package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func _printf(args ...interface{}) {
	s := ""
	for _, x := range args {
		s += fmt.Sprintf("%# v", pretty.Formatter(x))
	}
	fmt.Println(s)
}

func TestFindNumber(t *testing.T) {
	for _, test := range []struct {
		strBefore, strAfter string
		number, start, end  int
		err                 bool
	}{
		{"01", "", 1, 0, 1, false},
		{"a11yylmao", "ayylmao", 11, 1, 2, false},
		{"___asdfx_+++", "___asdfx_+++", 0, -1, -1, true},
		{"72test", "test", 72, 0, 1, false},
		{"\x0315123", "\x03123", 15, 1, 2, false},
		{"\x031500:32\x0f \x0312NOTICE: *** Looking up your hostname...\x0f",
			"\x0300:32\x0f \x0312NOTICE: *** Looking up your hostname...\x0f", 15, 1, 2, false},
	} {
		n, s, e, err := findNumber([]rune(test.strBefore), 1)
		var after string
		if test.start == -1 && test.end == -1 {
			after = test.strBefore
		} else if s >= 0 && e >= 0 && s < len(test.strBefore) && e < len(test.strBefore) {
			after = test.strBefore[:s] + test.strBefore[e+1:]
		} else {
			after = fmt.Sprintf("slice out of bounds: start: %d end: %d", s, e)
		}

		if n != test.number || s != test.start || e != test.end || (err != nil) != test.err || after != test.strAfter {
			fmt.Printf("expected: after: %#v number: %#v start: %#v end: %#v err: %#v\n",
				test.strAfter, test.number, test.start, test.end, test.err)
			fmt.Printf("actual:   after: %#v number: %#v start: %#v end: %#v err: %#v\n\n",
				after, n, s, e, err)
			t.Fail()
		}
	}
}

type richtext struct {
	str    string  // stripped of all control characters
	styles [][]int // style type, start offset, end offset, color value
}

func TestParseString(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected *richtext
	}{
		{
			input: "\x0313,3test",
			expected: &richtext{
				str: "test",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 4, colorPaletteWindows[13]},
					{TextEffectBackgroundColor, 0, 4, colorPaletteWindows[3]},
				},
			},
		},
		{
			input: string(fmtItalic) + "this" + string(fmtReset) + " is a " + string(fmtBold) + "\x034t\x037e\x038s\x033t " + string(fmtUnderline) + "https://" + string(fmtReset),
			expected: &richtext{
				str: "this is a test https://",
				styles: [][]int{
					{TextEffectItalic, 0, 4},
					{TextEffectBold, 10, 23},
					{TextEffectForegroundColor, 10, 11, colorPaletteWindows[4]},
					{TextEffectForegroundColor, 11, 12, colorPaletteWindows[7]},
					{TextEffectForegroundColor, 12, 13, colorPaletteWindows[8]},
					{TextEffectForegroundColor, 13, 23, colorPaletteWindows[3]},
					{TextEffectUnderline, 15, 23},
				},
			},
		},
		{
			input: "\x0313,3test\x0f and the rest of this should be unstyled",
			expected: &richtext{
				str: "test and the rest of this should be unstyled",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 4, colorPaletteWindows[13]},
					{TextEffectBackgroundColor, 0, 4, colorPaletteWindows[3]},
				},
			},
		},
		{
			input: string(fmtItalic) + "italic" + string(fmtItalic) + string(fmtBold) + "bold" + string(fmtBold) + string(fmtUnderline) + "underline" + string(fmtUnderline),
			expected: &richtext{
				str: "italicboldunderline",
				styles: [][]int{
					{TextEffectItalic, 0, 6},
					{TextEffectBold, 6, 10},
					{TextEffectUnderline, 10, 19},
				},
			},
		},
		{
			input: string(fmtItalic) + "italic" + string(fmtBold) + "bold" + string(fmtItalic) + string(fmtBold) + string(fmtUnderline) + "underline" + string(fmtUnderline),
			expected: &richtext{
				str: "italicboldunderline",
				styles: [][]int{
					{TextEffectItalic, 0, 10},
					{TextEffectBold, 6, 10},
					{TextEffectUnderline, 10, 19},
				},
			},
		},
		{
			input: color("test", Purple),
			expected: &richtext{
				str: "test",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 4, colorPaletteWindows[Purple]},
				},
			},
		},
		{
			input: color("test", White, Purple),
			expected: &richtext{
				str: "test",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 4, colorPaletteWindows[White]},
					{TextEffectBackgroundColor, 0, 4, colorPaletteWindows[Purple]},
				},
			},
		},
		{
			input: italic("this") + " is a " + bold("\x034t\x037e\x038s\x033t "+underline("https://")),
			expected: &richtext{
				str: "this is a test https://",
				styles: [][]int{
					{TextEffectItalic, 0, 4},
					{TextEffectBold, 10, 23},
					{TextEffectForegroundColor, 10, 11, colorPaletteWindows[4]},
					{TextEffectForegroundColor, 11, 12, colorPaletteWindows[7]},
					{TextEffectForegroundColor, 12, 13, colorPaletteWindows[8]},
					{TextEffectForegroundColor, 13, 23, colorPaletteWindows[3]},
					{TextEffectUnderline, 15, 23},
				},
			},
		},
		{
			input: color("gray", LightGrey),
			expected: &richtext{
				str: "gray",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 4, colorPaletteWindows[15]},
				},
			},
		},
		{
			input: "\x031500:32\x0f \x0312NOTICE: *** Looking up your hostname...\x0f",
			expected: &richtext{
				str: "00:32 NOTICE: *** Looking up your hostname...",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 5, colorPaletteWindows[15]},
					{TextEffectForegroundColor, 6, 45, colorPaletteWindows[12]},
				},
			},
		},
		{
			input: "\x031世界",
			expected: &richtext{
				str: "世界",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 2, colorPaletteWindows[1]},
				},
			},
		},
		{
			input: "\x031世\x032界",
			expected: &richtext{
				str: "世界",
				styles: [][]int{
					{TextEffectForegroundColor, 0, 1, colorPaletteWindows[1]},
					{TextEffectForegroundColor, 1, 2, colorPaletteWindows[2]},
				},
			},
		},
	} {
		text, styles := parseString(test.input)
		actual := &richtext{text, styles}
		if !reflect.DeepEqual(actual, test.expected) {
			fmt.Println("expected:")
			_printf(test.expected)
			fmt.Println("\nactual:")
			_printf(actual)
			t.Fail()
		}
	}
}
