package main

import (
	"github.com/lxn/walk"
)

func globalKeyHandler(key walk.Key) {

	// shift+esc: toggle borderless window
	if key == walk.KeyEscape && walk.ShiftDown() {
		mw.ToggleBorder()
	}

	if walk.ControlDown() {
		switch key {
		case walk.KeyTab:
			// ctrl+tab:       next tab (move right)
			// ctrl+shift+tab: previous tab (move left)
			max := tabWidget.Pages().Len() - 1
			if max < 1 {
				return
			}
			index := tabWidget.CurrentIndex()
			if walk.ShiftDown() {
				index -= 1
				if index < 0 {
					index = max
				}
			} else {
				index += 1
				if index > max {
					index = 0
				}
			}
			tabWidget.SetCurrentIndex(index)

		case walk.KeyQ:
			// ctrl+q: exit application
			exit()

		case walk.KeyT:
			// ctrl+t: open a new empty tab
			// FIXME(tso): limit 1 empty tab at any given time
			newEmptyServerTab()

		case walk.KeyF4, walk.KeyW:
			// ctrl+f4: close current tab
			// ctrl+w:  close current tab

			// FIXME(tso): resolve commandContext/tabWithContext
			cmdctx := &commandContext{}
			ctx := tabMan.Find(currentTabFinder)
			if ctx == nil {
				panic("current tab is not in tabManager!")
			}
			cmdctx.servConn = ctx.servConn
			cmdctx.servState = ctx.servState
			cmdctx.chanState = ctx.chanState
			cmdctx.pmState = ctx.pmState
			cmdctx.tab = ctx.tab.(tabWithTextBuffer)

			closeCmd(cmdctx)
		}
	} else {
		switch key {
		case walk.KeyF2:
			// f2: increase transparency (sets transparency enabled)
			mw.SetTransparency(-16)
		case walk.KeyF3:
			// f3: enable/disable transparency
			mw.ToggleTransparency()
		case walk.KeyF4:

			// alt+f4: exit application
			if walk.AltDown() {
				exit()
			}

			// f4: decrease transparency (sets transparency enabled)
			mw.SetTransparency(16)
		}

	}

}

func insertCharacter(key walk.Key) rune {
	if walk.ControlDown() {
		switch key {
		case walk.KeyK:
			// ctrl+k: insert color code
			return fmtColor
		case walk.KeyB:
			// ctrl+k: insert bold (toggle)
			return fmtBold
		case walk.KeyI:
			// ctrl+k: insert italic (toggle)
			return fmtItalic
		case walk.KeyU:
			// ctrl+k: insert underline (toggle)
			return fmtUnderline
		case walk.KeyS:
			// ctrl+k: insert strikethrough (toggle)
			return fmtStrikethrough
		case walk.Key0, walk.KeyNumpad0:
			// ctrl+0: insert reset formatting
			return fmtReset

			// TODO(tso): fmtReverse

		}
	}
	return 0
}
