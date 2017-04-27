package main

import (
	"log"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var tabWidget *walk.TabWidget
var mw *walk.MainWindow

func main() {
	MainWindow{
		AssignTo: &mw,
		Title:    "IRC",
		MinSize:  Size{480, 640},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			TabWidget{
				AssignTo: &tabWidget,
			},
		},
	}.Create()

	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)

	mw.WindowBase.SetFont(font)

	tabWidget.SetPersistent(true)

	cfg := getClientConfig()

	var tb *walk.TextEdit

	l := &tsoLogger{}
	l.LogFn = func(str string) {
		tb.AppendText(str + "\r\n")
	}

	logging.SetLogger(l)

	p, err := walk.NewTabPage()
	checkErr(err)
	p.SetTitle(cfg.Host)
	v := walk.NewVBoxLayout()
	p.SetLayout(v)
	b := NewBuilder(p)
	TextEdit{
		MinSize:    Size{480, 580},
		AssignTo:   &tb,
		ReadOnly:   true,
		Persistent: true,
	}.Create(b)
	tabWidget.Pages().Add(p)
	checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(p)))
	tabWidget.SaveState()

	servConn := newServerConnection(cfg)
	servConn.connect()

	mw.Run()
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
