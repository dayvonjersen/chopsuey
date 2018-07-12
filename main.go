package main

import (
	"fmt"
	"log"
	"os"
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
		log.Fatalln("nice job breaking it hero.\n\n\ngit checkout master -- config.json")
		title := "error parsing config.json"
		msg := err.Error()
		icon := walk.MsgBoxIconError

		if os.IsNotExist(err) {
			title = "config.json not found"
			msg = "default configuration loaded"
			err = writeClientConfig()
			if err == nil {
				cwd, err2 := os.Getwd()
				checkErr(err2)
				msg += " and a new config.json file has been written in " + cwd
				icon = walk.MsgBoxIconInformation
			} else {
				msg += " but an error was encountered while trying to write the file: " + err.Error()
			}
			walk.MsgBox(mw, title, msg, icon)
		}
	}

	// FIXME(tso): empty tab is a nightmare holy fuck
	if len(clientCfg.AutoConnect) == 0 {
		servState := &serverState{
			connState: CONNECTION_EMPTY,
			user: &userState{
				nick: "nobody",
			},
			channels: map[string]*channelState{},
			privmsgs: map[string]*privmsgState{},
		}
		emptyView := NewServerTab(&serverConnection{}, servState)
		servState.tab = emptyView
		mw.WindowBase.Synchronize(func() {
			helpCmd(&commandContext{tab: emptyView})
		})
		servers = append(servers, servState)
	}

	// FIXME(tso): abstract opening a new server connection/tab
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

		// FIXME(tso): just use the logic from /server
		if cfg.Host != "" && cfg.Port != 0 && cfg.Nick != "" {
			go servConn.Connect(servState)
		}
	}

	mw.Run()
}
