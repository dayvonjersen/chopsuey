package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

const (
	CHATLOG_DIR     = "./chatlogs/"
	SCREENSHOTS_DIR = "./screenshots/"
	SCRIPTS_DIR     = "./scripts/"
	THEMES_DIR      = "./themes/"

	CONNECT_RETRIES        = 1
	CONNECT_RETRY_INTERVAL = time.Second
	CONNECT_TIMEOUT        = time.Second * 30
)

var (
	mw        *myMainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem
	systray   *walk.NotifyIcon

	mainWindowFocused bool = true // start focused because windows
	mainWindowHidden  bool = false
)

type myMainWindow struct {
	*walk.MainWindow
	textColor, bgColor walk.Color
}

func (mw *myMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_DRAWITEM {
		// use foreground color in statusBar
		item := (*win.DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		// log.Printf("got WM_DRAWITEM, item: % #v lParam: %v", pretty.Formatter(item), lParam)
		if item.HwndItem == mw.StatusBar().Handle() && item.ItemState <= 1 {
			win.SetTextColor(item.HDC, rgb2COLORREF(globalForegroundColor))
			textptr := (*uint16)(unsafe.Pointer(item.ItemData))
			text := win.UTF16PtrToString(textptr)
			textlen := int32(len(text))
			// log.Printf("text: % #v", pretty.Formatter(text))
			win.TextOut(item.HDC, item.RcItem.Left+20 /*16px icon size + 4px padding*/, item.RcItem.Top, textptr, textlen)
			return win.TRUE
		}
	}

	if msg == win.WM_SYSCOMMAND {
		// minimize/close to tray
		if wParam == win.SC_MINIMIZE || wParam == win.SC_CLOSE {
			win.ShowWindow(mw.Handle(), win.SW_HIDE)
			mainWindowHidden = true
			return 0
		}
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}

// show me the way-ay out
func exit() {
	// FIXME(tso): even cleaner shutdown
	// TODO(tso): send QUIT to all active server connections
	checkErr(mw.Close())
	systray.Dispose()
	os.Exit(1)
}

func main() {
	runtime.LockOSThread()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		exit()
	}()

	defer func() {
		if x := recover(); x != nil {
			printf(mw)
			panic(x)
		}
	}()

	logging.SetLogger(&debugLogger{})

	clientState = &_clientState{
		connections: []*serverConnection{},
		servers:     []*serverState{},
		tabs:        []tab{},
		mu:          &sync.Mutex{},
	}

	mw = new(myMainWindow)
	MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "chopsuey IRC " + VERSION_STRING,
		Layout: VBox{
			MarginsZero: true,
			SpacingZero: true,
		},
		Children: []Widget{},
		StatusBarItems: []StatusBarItem{
			StatusBarItem{
				AssignTo: &statusBar,
				Text:     "not connected to any networks...",
			},
		},
	}.Create()
	walk.InitWrapperWindow(mw)

	//
	// transparency:
	//
	// required:
	// win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.WS_EX_CONTROLPARENT|win.WS_EX_LAYERED|win.WS_EX_STATICEDGE)
	//
	// entire window, 50% transparent
	// SetLayeredWindowAttributes(mw.Handle(), 0, 0xff * 50, LWA_ALPHA)
	//
	// chromakey
	// NOTE(tso): no
	// SetLayeredWindowAttributes(mw.Handle(), 0xf0f0f0, 0, LWA_COLORKEY)

	//
	// ｂ ｏ ｎ ｅ ｌ ｅ ｓ ｓ
	//
	//win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.WS_EX_CONTROLPARENT|((win.WS_OVERLAPPEDWINDOW&(^win.WS_THICKFRAME))&(^win.WS_BORDER)))
	win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.WS_EX_CONTROLPARENT|win.WS_EX_STATICEDGE)
	win.ShowWindow(mw.Handle(), win.SW_NORMAL)

	var err error
	tabWidget, err = walk.NewTabWidgetWithStyle(mw, win.TCS_MULTILINE)
	checkErr(err)
	tabWidget.SetPersistent(false)

	mw.Children().Add(tabWidget)

	mw.SetBounds(walk.Rectangle{
		X:      1536,
		Y:      0,
		Width:  384,
		Height: 1050,
	})

	ico, err := walk.NewIconFromFile("chopsuey.ico")
	checkErr(err)
	mw.SetIcon(ico)
	SetStatusBarIcon("chopsuey.ico")

	systray, err = walk.NewNotifyIcon(mw.Handle())
	checkErr(err)
	defer systray.Dispose()
	systray.SetIcon(ico)
	systray.SetVisible(true)
	systray.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mainWindowHidden {
				win.ShowWindow(mw.Handle(), win.SW_NORMAL)
			} else {
				win.ShowWindow(mw.Handle(), win.SW_HIDE)
			}
			mainWindowHidden = !mainWindowHidden
		}
	})
	SetSystrayContextMenu()

	font, err := walk.NewFont("Hack", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)

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
	mw.SizeChanged().Attach(func() {
		for _, t := range clientState.tabs {
			switch t.(type) {
			case *tabServer:
				t := t.(*tabServer)
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			case *tabChannel:
				t := t.(*tabChannel)
				t.Resize()
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			case *tabPrivmsg:
				t := t.(*tabPrivmsg)
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			}
		}
	})

	clientState.cfg, err = getClientConfig()
	if err != nil {
		log.Println("error parsing config.json", err)
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
		SetStatusBarIcon("res/msg_error.ico")
		SetStatusBarText("error parsing config.json")
		/*}

		// XXX TEMPORARY SECRETARY
		for i := 0; i < 1; i++ {
			emptyTab := NewServerTab(&serverConnection{}, &serverState{
				networkName: "tab " + strconv.Itoa(i),
				user:        &userState{nick: "tso"},
			})

			mw.WindowBase.Synchronize(func() {
				paletteCmd(&commandContext{tab: emptyTab})
			})
		}
		*/

	} else {
		if clientState.cfg.Theme != "" {
			applyTheme(clientState.cfg.Theme)
		}
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
			servConn = NewServerConnection(servState,
				func(nickservPASSWORD string, autojoin []string) func() {
					return func() {
						if nickservPASSWORD != "" {
							servConn.conn.Privmsg("NickServ", "IDENTIFY "+nickservPASSWORD)
						}
						for _, channel := range autojoin {
							servConn.conn.Join(channel)
						}
					}
				}(cfg.NickServPASSWORD, cfg.AutoJoin),
			)
			clientState.mu.Lock()
			servView := NewServerTab(servConn, servState)
			clientState.mu.Unlock()
			servState.tab = servView
			servConn.Connect(servState)
		}
	}
	/**/

	mw.Run()
}

type debugLogger struct{}

func (l *debugLogger) Debug(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Info(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Warn(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Error(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }
