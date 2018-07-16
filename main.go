package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

var VERSION_STRING = fmt.Sprintf("v%.53f", 2.0/3.0)

const (
	CHATLOG_DIR = "./chatlogs/"
	SCRIPTS_DIR = "./scripts/"
	THEMES_DIR  = "./themes/"

	CONNECT_RETRIES        = 1
	CONNECT_RETRY_INTERVAL = time.Second
	CONNECT_TIMEOUT        = time.Second * 30
)

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	mainWindowFocused bool = true // start focused because windows
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

	userTheme, err := loadPaletteFromFile("zenburn")
	checkErr(err)
	loadColorPalette(userTheme[:16])
	bg := userTheme[16]
	r, g, b := byte((bg>>16)&0xff), byte((bg>>8)&0xff), byte(bg&0xff)
	brush, err := walk.NewSolidColorBrush(walk.RGB(r, g, b))
	checkErr(err)
	defer brush.Dispose()
	mw.SetBackground(brush)
	tabWidget.SetBackground(brush)
	sb := mw.StatusBar()
	sb.SetBackground(brush)
	fg := userTheme[17]
	colorref := win.COLORREF(fg&0xff<<16 | fg&0xff00 | fg&0xff0000>>16)
	win.SetTextColor(win.GetDC(mw.Handle()), colorref)
	win.SetTextColor(win.GetDC(tabWidget.Handle()), colorref)
	win.SetTextColor(win.GetDC(sb.Handle()), colorref)

	globalBackgroundColor = bg
	globalForegroundColor = fg

	tabWidget.SetPersistent(false)

	// NOTE(tso): contrary to what the name of this event publisher implies
	//            CurrentIndexChanged() fires every time you Insert() or Remove()
	//            a TabPage regardless of whether the CurrentIndex() actually
	//            changed.
	//
	//            and you *have* to set the CurrentIndex() again when you Add(),
	//            Insert() or Remove() for everything to draw correctly
	//
	//            e.g.
	//            tabs: [0 1 2 3], currentIndex == 1
	//            Add()
	//            tabs: [0 1 2 3 4], currentIndex still == 1
	//            CurrentIndexChanged fires
	//              uhhhhhhhhhhhhhhhhhhh
	//
	//            at least that's what I think is happening, probably wrong about
	//            something
	// -tso 7/14/2018 11:29:50 PM

	var currentFocusedTab tab
	tabWidget.CurrentIndexChanged().Attach(func() {
		currentTab := clientState.CurrentTab()
		if currentFocusedTab != currentTab {
			currentFocusedTab = currentTab
			currentTab.Focus()
		}
	})
	mw.Activating().Attach(func() {
		mainWindowFocused = true
		// always call Focus() when window regains focus
		clientState.CurrentTab().Focus()
	})
	mw.Deactivating().Attach(func() {
		mainWindowFocused = false
	})

	clientState = &_clientState{
		connections: []*serverConnection{},
		servers:     []*serverState{},
		tabs:        []tab{},
	}
	clientState.cfg, err = getClientConfig()
	if err != nil {
		log.Println("error parsing config.json", err)
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
		statusBar.SetText("error parsing config.json")
	} else {
		for _, cfg := range clientState.cfg.AutoConnect {
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
				if cfg.NickServPASSWORD != "" {
					servConn.conn.Privmsg("NickServ", "IDENTIFY "+cfg.NickServPASSWORD)
				}
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
