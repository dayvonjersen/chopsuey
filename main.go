package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

	TRANSPARENCY_DEFAULT_ALPHA = 0xb4 // a nice default value: ~70% opaque
)

var (
	mw        *myMainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem
	systray   *walk.NotifyIcon

	mainWindowFocused = true // start focused because windows (we don't get WM_ACTIVATE or WM_CREATE reliably)
	mainWindowHidden  = false

	clientCfg *clientConfig
	tabMan    *tabManager
)

func exit() {
	// TODO(tso): send QUIT to all active server connections
	close(tabMan.destroy)
	checkErr(mw.Close())
	systray.Dispose()
	os.Exit(1)
}

func main() {
	// ...
	var err error

	// handle ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; exit() }()

	// print the state of the whole window on panic to catch winapi-related bugs
	defer func() {
		if x := recover(); x != nil {
			printf(mw)
			panic(x)
		}
	}()

	// goirc logging...
	logging.SetLogger(&debugLogger{})

	// tab management!
	tabMan = newTabManager()

	// create the window
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

	//  required for transparency:
	mw.alpha = TRANSPARENCY_DEFAULT_ALPHA
	win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.WS_EX_CONTROLPARENT|win.WS_EX_STATICEDGE|win.WS_EX_LAYERED)
	SetLayeredWindowAttributes(mw.Handle(), 0, 0xff, LWA_ALPHA) // have to do this for the window to draw if aero is disabled
	win.ShowWindow(mw.Handle(), win.SW_NORMAL)

	// create tab widget
	tabWidget, err = walk.NewTabWidgetWithStyle(mw, win.TCS_MULTILINE)
	checkErr(err)
	tabWidget.SetPersistent(true)
	mw.Children().Insert(0, tabWidget)

	// set main window size and position
	// TODO(tso): configurable? save position? optional?
	mw.SetBounds(walk.Rectangle{
		X:      1536,
		Y:      0,
		Width:  384,
		Height: 1050,
	})

	// set default font
	// TODO(tso): configurable font
	font, err := walk.NewFont("Hack", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)

	// set icons
	ico, err := walk.NewIconFromFile("chopsuey.ico")
	checkErr(err)
	mw.SetIcon(ico)
	SetStatusBarIcon("chopsuey.ico")

	// create system tray
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

	//
	// f o c u s
	//
	var currentFocusedTab tab
	tabWidget.CurrentIndexChanged().Attach(func() {
		ctx := tabMan.Find(currentTabFinder)
		if ctx == nil {
			return
		}
		currentTab := ctx.tab
		if currentFocusedTab != currentTab {
			currentFocusedTab = currentTab
			currentTab.Focus()
		}
	})
	mw.Activating().Attach(func() {
		mainWindowFocused = true
		ctx := tabMan.Find(currentTabFinder)
		if ctx == nil {
			return
		}
		ctx.tab.Focus()
	})
	mw.Deactivating().Attach(func() { mainWindowFocused = false })

	// resize
	// FIXME(tso): @resizing
	mw.SizeChanged().Attach(func() {
		for _, ctx := range tabMan.FindAll(allTabsFinder) {
			t := ctx.tab
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

	// config.json
	clientCfg, err = getClientConfig()
	if err != nil {
		msg := "error parsing config.json"
		log.Println(msg, err)
		mw.Synchronize(func() {
			walk.MsgBox(mw, msg, err.Error(), walk.MsgBoxIconError)
			SetStatusBarIcon("res/msg_error.ico")
			SetStatusBarText(msg)
			// TODO(tso): create+load+write default clientConfig
			newEmptyServerTab()
		})
	} else {
		if clientCfg.Theme != "" {
			if err := applyTheme(clientCfg.Theme); err != nil {
				walk.MsgBox(mw, err.Error(), err.Error(), walk.MsgBoxIconError)
			}
		}
		if len(clientCfg.AutoConnect) == 0 {
			mw.Synchronize(func() {
				newEmptyServerTab()
			})
		} else {
			go func() {
				// autojoin
				for _, cfg := range clientCfg.AutoConnect {
					// TODO(tso): abstract opening a new server connection/tab
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
					index := tabMan.Len()
					ctx := tabMan.Create(&tabContext{servConn: servConn, servState: servState}, index)
					tab := newServerTab(servConn, servState)
					ctx.tab = tab
					servState.tab = tab
					servConn.Connect(servState)
				}
			}()
		}
	}

	mw.Run()
}

// goirc logging...
type debugLogger struct{}

func (l *debugLogger) Debug(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Info(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Warn(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Error(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }

// here be dragons
type myMainWindow struct {
	*walk.MainWindow

	transparent bool
	alpha       int32

	borderless bool
}

func (mw *myMainWindow) SetTransparency(amt int32) {
	mw.alpha += amt
	if mw.alpha < 0 {
		mw.alpha = 0
	}
	if mw.alpha > 0xff {
		mw.alpha = 0xff
	}
	if mw.alpha == 0xff {
		mw.transparent = false
	} else {
		mw.transparent = true
	}
	SetLayeredWindowAttributes(mw.Handle(), 0, mw.alpha, LWA_ALPHA)
}

func (mw *myMainWindow) ToggleTransparency() {
	if mw.transparent {
		SetLayeredWindowAttributes(mw.Handle(), 0, 0xff, LWA_ALPHA)
	} else {
		if mw.alpha == 0xff {
			mw.alpha = TRANSPARENCY_DEFAULT_ALPHA
		}
		SetLayeredWindowAttributes(mw.Handle(), 0, mw.alpha, LWA_ALPHA)
	}
	win.ShowWindow(mw.Handle(), win.SW_NORMAL)
	mw.transparent = !mw.transparent
}

func (mw *myMainWindow) ToggleBorder() {
	if mw.borderless {
		mw.StatusBar().SetVisible(true)
		win.SetWindowLong(mw.Handle(), win.GWL_STYLE, win.WS_OVERLAPPEDWINDOW)
	} else {
		mw.StatusBar().SetVisible(false)
		// NOTE(tso): SFML does this:
		//win.SetWindowLongPtr(mw.Handle(), win.GWL_STYLE, uintptr(win.WS_VISIBLE|win.WS_POPUP))
		win.SetWindowLong(mw.Handle(), win.GWL_STYLE, win.WS_OVERLAPPEDWINDOW&^win.WS_THICKFRAME&^win.WS_BORDER)
	}
	win.ShowWindow(mw.Handle(), win.SW_NORMAL)
	mw.borderless = !mw.borderless
}

func (mw *myMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_DRAWITEM:
		// use foreground color for statusBar text
		item := (*win.DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if item.HwndItem == mw.StatusBar().Handle() && item.ItemAction == 0 && item.ItemState&ODS_SELECTED == ODS_SELECTED {

			var iconOffset int32 = 20 /*16px icon size + 4px padding*/
			item.RcItem.Left += iconOffset
			item.RcItem.Right += iconOffset

			bg := rgb2COLORREF(globalBackgroundColor)

			/* slow, but I'm done with figuring out winapi bullshit right now. */
			for x := item.RcItem.Left; x < item.RcItem.Right; x++ {
				for y := item.RcItem.Top; y < item.RcItem.Bottom; y++ {
					win.SetPixel(item.HDC, x, y, bg)
				}
			}

			/*
				width := item.RcItem.Right - item.RcItem.Left
				height := item.RcItem.Bottom - item.RcItem.Top
				bmpHeader := &win.BITMAPINFOHEADER{
				   BiWidth:    width,
				   BiHeight:   height,
				   BiPlanes:   1,
				   BiBitCount: 32,
				   BiClrUsed:  uint32(bg),
				}
				bmpHeader.BiSize = uint32(unsafe.Sizeof(bmpHeader))
				whatever := 0
				dontcare := unsafe.Pointer(&whatever)
				bmp := win.CreateDIBSection(item.HDC, bmpHeader, 0, &dontcare, 0, 0)
				win.SetDIBits(item.HDC, bmp, 0, 0, nil, nil, 0)
			*/

			textptr := (*uint16)(unsafe.Pointer(item.ItemData))
			text := win.UTF16PtrToString(textptr)
			textlen := int32(len(text))
			win.SetTextColor(item.HDC, rgb2COLORREF(globalForegroundColor))
			win.TextOut(item.HDC, item.RcItem.Left, item.RcItem.Top, textptr, textlen)
			return win.TRUE
		}

	case win.WM_SYSCOMMAND:
		// minimize/close to tray
		if wParam == win.SC_MINIMIZE || wParam == win.SC_CLOSE {
			win.ShowWindow(mw.Handle(), win.SW_HIDE)
			mainWindowHidden = true
			return 0
		}
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}
