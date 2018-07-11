package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/kr/pretty"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const CHATLOG_DIR = "./chatlogs/"

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	clientCfg *clientConfig

	connections []*serverConnection
	servers     []*serverState
	tabs        []tabView
)

/*
might need to remember how to do this in the future:

type myMainWindow struct {
	*walk.MainWindow
}

func (mw *myMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_ACTIVATE {
		log.Println("got WM_ACTIVATE")
		focusCurrentTab()
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}

func main() {
	mw = new(myMainWindow)
	MainWindow{
		AssignTo: &mw.MainWindow,
		// ...
	}.Create()
	walk.InitWrapperWindow(mw)
}
*/

var mainWindowFocused bool = true // start focused because windows
/*
this is better but windows
func mainWindowFocused() bool {
	return mw.Handle() == win.GetFocus()
}
*/

/*
var (
	clientState *clientState
)

type clientState struct {
	cfg *clientConfig

	connections []*serverConnection
	servers     []*serverState
	tabs        []tabView
}
*/

func main() {
	runtime.LockOSThread()

	MainWindow{
		AssignTo: &mw,
		Title:    "chopsuey IRC v0.5",
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			TabWidget{
				AssignTo: &tabWidget,
			},
		},
		StatusBarItems: []StatusBarItem{
			StatusBarItem{
				AssignTo: &statusBar,
				Text:     "not connected to any networks...",
			},
		},
	}.Create()

	mw.SetBounds(walk.Rectangle{
		X:      1536,
		Y:      0,
		Width:  384,
		Height: 1050,
	})

	ico, err := walk.NewIconFromFile("chopsuey.ico")
	checkErr(err)
	mw.SetIcon(ico)

	tabWidget.SetPersistent(true)

	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)

	// debug log
	const DEBUG_WRITE_TO_FILE = false

	l := &tsoLogger{}
	if DEBUG_WRITE_TO_FILE {
		filename := "./log/" + time.Now().Format("20060102150405.999999999") + ".log"
		f, err := os.Create(filename)
		checkErr(err)
		defer f.Close()
		l.LogFn = func(msg string) {
			io.WriteString(f, msg+"\n")
		}
	} else {
		l.LogFn = func(msg string) {
			fmt.Println(msg)
		}
	}
	logging.SetLogger(l)

	focusCurrentTab := func() {
		index := tabWidget.CurrentIndex()
		for _, t := range tabs {
			if t.Index() == index {
				t.Focus()
				return
			}
		}
	}

	mw.Activating().Attach(func() {
		mainWindowFocused = true
		focusCurrentTab()
	})
	mw.Deactivating().Attach(func() {
		mainWindowFocused = false
	})
	tabWidget.CurrentIndexChanged().Attach(focusCurrentTab)

	clientCfg, err = getClientConfig()
	if err != nil {
		log.Println("error parsing config.json", err)
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
		statusBar.SetText("error parsing config.json")
	} else {
		for _, cfg := range clientCfg.AutoConnect {
			servState := &serverState{
				connState:   CONNECTION_EMPTY,
				hostname:    cfg.Host,
				port:        cfg.Port,
				ssl:         cfg.Ssl,
				networkName: cfg.ServerString(),
				user: &userState{
					nick: cfg.Nick,
				},
				channels: map[string]*channelState{},
				privmsgs: map[string]*privmsgState{},
			}
			var servConn *serverConnection
			servConn = NewServerConnection(servState, func() {
				for _, channel := range cfg.AutoJoin {
					servConn.conn.Join(channel)
				}
			})
			servView := NewServerTab(servConn, servState)
			servState.tab = servView
			servConn.Connect(servState)
		}
	}

	mw.Run()
}

type tsoLogger struct {
	LogFn func(string)
}

func (l *tsoLogger) Debug(f string, a ...interface{}) { l.LogFn(fmt.Sprintf(f, a...)) }
func (l *tsoLogger) Info(f string, a ...interface{})  { l.LogFn(fmt.Sprintf(f, a...)) }
func (l *tsoLogger) Warn(f string, a ...interface{})  { l.LogFn(fmt.Sprintf(f, a...)) }
func (l *tsoLogger) Error(f string, a ...interface{}) { l.LogFn(fmt.Sprintf(f, a...)) }

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func now() string {
	return time.Now().Format(clientCfg.TimeFormat)
}

func printf(args ...interface{}) {
	s := ""
	for _, x := range args {
		s += fmt.Sprintf("%# v", pretty.Formatter(x))
	}
	log.Print(s)
}
