package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/fluffle/goirc/logging"
	"github.com/kr/pretty"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem
)

func main() {
	MainWindow{
		AssignTo: &mw,
		Title:    "IRC",
		MinSize:  Size{480, 680},
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

	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)

	mw.WindowBase.SetFont(font)

	tabWidget.SetPersistent(true)
	tabWidget.CurrentIndexChanged().Attach(func() {
		children := tabWidget.Pages().At(tabWidget.CurrentIndex()).Children()
		for i := 0; i < children.Len(); i++ {
			child := children.At(i)
			if reflect.TypeOf(child).String() == "*walk.LineEdit" {
				lineEdit := child.(*walk.LineEdit)
				if lineEdit.ReadOnly() == false {
					lineEdit.SetFocus()
					break
				}
			}
		}
	})

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

	statusBar.SetText("connecting to " + cfg.ServerString() + "...")
	servConn := newServerConnection(cfg)
	servConn.connect()

	mw.Run()
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func printf(args ...interface{}) {
	s := ""
	for _, x := range args {
		s += fmt.Sprintf("%# v", pretty.Formatter(x))
	}
	log.Print(s)
}
