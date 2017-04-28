package main

import (
	"fmt"
	"log"
	"reflect"

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
