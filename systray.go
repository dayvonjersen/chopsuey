package main

import (
	"log"
	"os/exec"
	"sync"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var mu = &sync.Mutex{}

func SetSystrayContextMenu() {
	return
	mu.Lock()
	defer mu.Unlock()

	type menuItem struct {
		separator bool

		text string
		fn   func()
	}

	menu := make([]menuItem, len(clientState.tabs))

	i := len(clientState.tabs) - 1
	for _, t := range clientState.tabs {
		tabTitle := t.Title()
		_, split := t.(*tabServer)
		// FIXME(tso): have to do this because tab creation is happening in a mw.Synchronize
		//             and t.Index() is -1 until the tab actually gets created...
		idx := t.Index()
		if idx == -1 {
			idx = i
			i--
		}
		menu[idx] = menuItem{
			separator: split,
			text:      tabTitle,
			// FIXME(tso): have to do this because tab creation is happening in a mw.Synchronize
			//             and t.Index() is -1 until the tab actually gets created...
			fn: func(t tab) func() {
				return func() {
					tabWidget.SetCurrentIndex(t.Index())
					mainWindowHidden = false
					win.ShowWindow(mw.Handle(), win.SW_NORMAL)
				}
			}(t),
		}
	}

	menu = append(menu,
		menuItem{
			separator: true,
			text:      "Settings",
			fn:        settingsDialog,
		},
		menuItem{
			text: "About",
			fn:   aboutDialog,
		},
		menuItem{
			text: "Help",
			fn: func() {
				if t, ok := clientState.CurrentTab().(tabWithTextBuffer); ok {
					ctx := &commandContext{tab: t}
					helpCmd(ctx)
				}
			},
		},
		menuItem{
			separator: true,
			text:      "Report Issue on GitHub",
			fn:        reportIssue,
		},
		menuItem{
			separator: true,
			text:      "Quit",
			fn:        exit,
		},
	)

	systray.ContextMenu().Actions().Clear()
	for _, item := range menu {
		if item.separator {
			systray.ContextMenu().Actions().Add(walk.NewSeparatorAction())
		}
		action := walk.NewAction()
		action.SetText(item.text)
		action.Triggered().Attach(item.fn)
		systray.ContextMenu().Actions().Add(action)
	}
}

func reportIssue() {
	url := "https://github.com/generaltso/chopsuey/issues/new"
	cmd := exec.Command("cmd", "/c", "start", url)
	if err := cmd.Run(); err != nil {
		log.Println("cmd /c start", url, "returned error:\n", err)
	}
}

func aboutDialog() {
	walk.MsgBox(mw, "About", "    chopsuey "+VERSION_STRING+`

    github.com/generaltso/chopsuey

    tso@teknik.io
    `, walk.MsgBoxOK)
}

func settingsDialog() {
	walk.MsgBox(mw, "Settings", "soon™", walk.MsgBoxOK)
}
