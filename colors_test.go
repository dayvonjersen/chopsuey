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
		{"00", "", 0, 0, 1, false},
		{"a11yylmao", "ayylmao", 11, 1, 2, false},
		{"___asdfx_+++", "___asdfx_+++", 0, -1, -1, true},
		{"72test", "test", 72, 0, 1, false},
	} {
		n, s, e, err := findNumber(test.strBefore)
		var after string
		if test.start == -1 && test.end == -1 {
			after = test.strBefore
		} else if s >= 0 && e >= 0 && s < len(test.strBefore) && e < len(test.strBefore) {
			after = test.strBefore[:s] + test.strBefore[e+1:]
		} else {
			after = fmt.Sprintf("slice out of bounds: start: %d end: %d", s, e)
		}

		if n != test.number || s != test.start || e != test.end || (err != nil) != test.err || after != test.strAfter {
			t.Errorf(
				"expected: %#v \nactual: after: %#v number: %#v err: %#v",
				test, after, n, err,
			)
		}
	}
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
					{styleForegroundColor, 0, 4, 0xff00ff},
					{styleBackgroundColor, 0, 4, 0x018000},
				},
			},
		},

		{
			input: fmtItalic + "this" + fmtReset + " is a " + fmtBold + "\x034t\x037e\x038s\x033t " + fmtUnderline + "https://" + fmtReset, //+ fmtUnderline + fmtRed + "g" + fmtOrange + "i" + fmtYellow + "t" + fmtGreen + "h" + fmtBlue + "u" + fmtTeal + "b" + fmtPurple + ".com" + fmtReset + "/generaltso/chopsuey\r\n\r\nkill me",
			expected: &richtext{
				str: "this is a test https://", //github.com/generaltso/chopsuey\r\n\r\nkill me",
				styles: [][]int{
					{styleItalic, 0, 4},
					{styleBold, 10, 23},
					{styleForegroundColor, 10, 11, colorPaletteWindows[4]},
					{styleForegroundColor, 11, 12, colorPaletteWindows[7]},
					{styleForegroundColor, 12, 13, colorPaletteWindows[8]},
					{styleForegroundColor, 13, 23, colorPaletteWindows[3]},
					{styleUnderline, 15, 23},
				},
			},
		},
	} {
		actual := parseString(test.input)
		if !reflect.DeepEqual(actual, test.expected) {
			fmt.Println("expected:")
			_printf(test.expected)
			fmt.Println("\nactual:")
			_printf(actual)
			t.Error("whoosp")
		}
	}
}
