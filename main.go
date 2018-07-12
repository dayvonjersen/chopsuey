package main

import (
	"fmt"
	"runtime"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	CHATLOG_DIR = "./chatlogs/"
	SCRIPTS_DIR = "./scripts/"
)

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	mainWindowFocused bool = true // start focused because windows

	clientCfg *clientConfig

	connections []*serverConnection
	servers     []*serverState
	tabs        []tabView
)

type debugLogger struct{}

func (l *debugLogger) Debug(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Info(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Warn(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Error(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }

func main() {
	runtime.LockOSThread()

	logging.SetLogger(&debugLogger{})

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

	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)

	tabWidget.SetPersistent(true)

	focusCurrentTab := func() {
		index := tabWidget.CurrentIndex()
		for _, t := range tabs {
			if t.Index() == index {
				t.Focus()
				return
			}
		}
	}

	tabWidget.CurrentIndexChanged().Attach(focusCurrentTab)
	mw.Activating().Attach(func() {
		mainWindowFocused = true
		focusCurrentTab()
	})
	mw.Deactivating().Attach(func() {
		mainWindowFocused = false
	})

	if clientCfg, err = getClientConfig(); err == nil {
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
	} else {
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
	}

	mw.Run()
}
