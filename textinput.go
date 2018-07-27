package main

import (
	"log"
	"strings"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

func newMyLineEdit(parent walk.Container) *MyLineEdit {
	le := new(MyLineEdit)
	checkErr(walk.InitWindow(
		le,
		parent,
		"EDIT",
		win.WS_CHILD|win.WS_TABSTOP|win.WS_VISIBLE|win.ES_AUTOHSCROLL,
		win.WS_EX_CLIENTEDGE,
	))
	le.tabComplete = &tabComplete{}
	return le
}

type MyLineEdit struct {
	walk.LineEdit

	msgHistory      []string
	msgHistoryIndex int
	tabComplete     *tabComplete
}

type tabComplete struct {
	Active  bool
	Entries []string
	Index   int
}

func (le *MyLineEdit) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_GETDLGCODE {
		if wParam == win.VK_TAB || wParam == win.VK_ESCAPE || wParam == win.VK_RETURN {
			return win.DLGC_WANTMESSAGE
		}
	}
	return le.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

func getCommandContext(t tabWithTextBuffer) *commandContext {
	// FIXME(tso): resolve commandContext/tabWithContext
	cmdctx := &commandContext{}
	ctx := tabMan.Find(identityFinder(t))
	if ctx == nil {
		panic("textInput owner tab is not in tabManager!")
	}
	cmdctx.servConn = ctx.servConn
	cmdctx.servState = ctx.servState
	cmdctx.chanState = ctx.chanState
	cmdctx.pmState = ctx.pmState
	cmdctx.tab = t
	return cmdctx
}

func NewTextInput(t tabWithTextBuffer) *MyLineEdit {
	var tabPage *walk.TabPage
	var sendFn func(string)
	switch t.(type) {
	case *tabServer:
		tabPage = t.(*tabServer).tabPage
		sendFn = func(str string) {
			clientError(t, "cannot send messages to a SERVER!!", "\nnot sent:", color(str, LightGrey))
		}
	case *tabChannel:
		tabPage = t.(*tabChannel).tabPage
		sendFn = t.(*tabChannel).Send
	case *tabPrivmsg:
		tabPage = t.(*tabPrivmsg).tabPage
		sendFn = t.(*tabPrivmsg).Send
	default:
		log.Panicf("unsupported type %T", t)
	}
	textInput := newMyLineEdit(tabPage)

	textInput.KeyDown().Attach(func(key walk.Key) {
		if r := insertCharacter(key); r != 0 {
			text := []rune(textInput.Text())
			s, e := textInput.TextSelection()
			text = append(text[:s], append([]rune{r}, text[e:]...)...)
			textInput.SetText(string(text))
			textInput.SetTextSelection(s+1, e+1)
		} else if key == walk.KeyReturn {
			text := textInput.Text()
			if len(text) < 1 {
				return
			}
			textInput.msgHistory = append(textInput.msgHistory, text)
			textInput.msgHistoryIndex = len(textInput.msgHistory) - 1
			if text[0] == '/' && len(text) > 1 {
				parts := strings.Split(text[1:], " ")
				cmd := parts[0]
				if cmd[0] == '/' {
					sendFn(text[1:])
				} else {
					var args []string
					if len(parts) > 1 {
						args = parts[1:]
					} else {
						args = []string{}
					}
					if cmdFn, ok := clientCommands[cmd]; ok {
						cmdFn(getCommandContext(t), args...)
					} else {
						clientError(t, "unrecognized command: ", cmd)
					}
				}
			} else {
				sendFn(text)
			}
			textInput.SetText("")
		} else if key == walk.KeyUp {
			if len(textInput.msgHistory) > 0 {
				text := textInput.msgHistory[textInput.msgHistoryIndex]
				textInput.SetText(text)
				textInput.SetTextSelection(len(text), len(text))
				textInput.msgHistoryIndex--
				if textInput.msgHistoryIndex < 0 {
					textInput.msgHistoryIndex = 0
				}
			}
		} else if key == walk.KeyDown {
			if len(textInput.msgHistory) > 0 {
				textInput.msgHistoryIndex++
				if textInput.msgHistoryIndex <= len(textInput.msgHistory)-1 {
					text := textInput.msgHistory[textInput.msgHistoryIndex]
					textInput.SetText(text)
					textInput.SetTextSelection(len(text), len(text))
				} else {
					textInput.SetText("")
					textInput.msgHistoryIndex = len(textInput.msgHistory) - 1
				}
			}
		}
	})

	textInput.KeyUp().Attach(func(key walk.Key) {
		if key == walk.KeyUp || key == walk.KeyDown {
			text := textInput.Text()
			textInput.SetTextSelection(len(text), len(text))
		}
	})

	textInput.KeyPress().Attach(globalKeyHandler)
	textInput.KeyPress().Attach(func(key walk.Key) {
		if key == walk.KeyUp || key == walk.KeyDown {
			text := textInput.Text()
			textInput.SetTextSelection(len(text), len(text))
		} else if key == walk.KeyTab && !walk.ControlDown() {
			text := strings.Split(textInput.Text(), " ")
			if textInput.tabComplete.Active {
				textInput.tabComplete.Index++
				if textInput.tabComplete.Index >= len(textInput.tabComplete.Entries) {
					textInput.tabComplete.Index = 0
				}
			} else {
				term := text[len(text)-1]
				res := []string{}
				ctx := getCommandContext(t)
				if ctx.chanState != nil {
					res = ctx.chanState.nickList.Search(term)
				}
				res = append(res, term)
				textInput.tabComplete = &tabComplete{
					Active:  true,
					Entries: res,
					Index:   0,
				}
			}
			text = append(text[:len(text)-1], textInput.tabComplete.Entries[textInput.tabComplete.Index])
			txt := strings.Join(text, " ")
			textInput.SetText(txt)
			textInput.SetTextSelection(len(txt), len(txt))
		} else {
			if textInput.tabComplete.Active {
				textInput.tabComplete = &tabComplete{}
			}
		}
	})
	return textInput
}
