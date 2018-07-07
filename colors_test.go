package main

import (
	"fmt"
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
			input: "\x0372,69test",
			expected: &richtext{
				str:      "test",
				fgColors: [][3]int{{72, 0, 4}},
				bgColors: [][3]int{{69, 0, 4}},
			},
		},
		{
			input: "\x0372,69te\x0fst",
			expected: &richtext{
				str:      "test",
				fgColors: [][3]int{{72, 0, 2}},
				bgColors: [][3]int{{69, 0, 2}},
			},
		},
	} {
		actual := parseString(test.input)
		failed := (actual.str != test.expected.str ||
			len(actual.fgColors) != len(test.expected.fgColors) ||
			len(actual.bgColors) != len(test.expected.bgColors) ||
			len(actual.bold) != len(test.expected.bold) ||
			len(actual.italic) != len(test.expected.italic) ||
			len(actual.underline) != len(test.expected.underline))
		if !failed {
			for i, fg := range actual.fgColors {
				if test.expected.fgColors[i][0] != fg[0] ||
					test.expected.fgColors[i][1] != fg[1] ||
					test.expected.fgColors[i][2] != fg[2] {
					failed = true
					break
				}
			}
			for i, bg := range actual.bgColors {
				if test.expected.bgColors[i][0] != bg[0] ||
					test.expected.bgColors[i][1] != bg[1] ||
					test.expected.bgColors[i][2] != bg[2] {
					failed = true
					break
				}
			}
			for i, bold := range actual.bold {
				if test.expected.bold[i][0] != bold[0] ||
					test.expected.bold[i][1] != bold[1] {
					failed = true
					break
				}
			}
			for i, italic := range actual.italic {
				if test.expected.italic[i][0] != italic[0] ||
					test.expected.italic[i][1] != italic[1] {
					failed = true
					break
				}
			}
			for i, underline := range actual.underline {
				if test.expected.underline[i][0] != underline[0] ||
					test.expected.underline[i][1] != underline[1] {
					failed = true
					break
				}
			}
		}

		if failed {
			fmt.Println("expected:")
			_printf(test.expected)
			fmt.Println("\nactual:")
			_printf(actual)
			t.Error("whoosp")
		}
	}
}
