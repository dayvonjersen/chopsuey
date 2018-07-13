package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	VERSION_STRING = "v0.5"

	CHATLOG_DIR = "./chatlogs/"
	SCRIPTS_DIR = "./scripts/"

	CONNECT_RETRIES        = 100
	CONNECT_RETRY_INTERVAL = time.Second
	CONNECT_TIMEOUT        = time.Second * 30
)

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	mainWindowFocused bool = true // start focused because windows

	clientCfg *clientConfig

	connections []*serverConnection
	servers     []*serverState
	tabs        []tab
)

func getCurrentTab() tab {
	index := tabWidget.CurrentIndex()
	for _, t := range tabs {
		if t.Index() == index {
			return t
		}
	}
	return nil
}

func getCurrentTabForServer(servState *serverState) tabWithInput {
	index := tabWidget.CurrentIndex()
	if servState.tab.Index() == index {
		return servState.tab
	}
	for _, ch := range servState.channels {
		if ch.tab.Index() == index {
			return ch.tab
		}
	}
	for _, pm := range servState.privmsgs {
		if pm.tab.Index() == index {
			return pm.tab
		}
	}
	return servState.tab
}

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
		Title:    "chopsuey IRC " + VERSION_STRING,
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
		getCurrentTab().Focus()
	}

	tabWidget.CurrentIndexChanged().Attach(focusCurrentTab)
	mw.Activating().Attach(func() {
		mainWindowFocused = true
		focusCurrentTab()
	})
	mw.Deactivating().Attach(func() {
		mainWindowFocused = false
	})

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
				networkName: serverAddr(cfg.Host, cfg.Port),
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
