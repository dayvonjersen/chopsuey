package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/kr/pretty"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	mw        *walk.MainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	clientCfg *clientConfig
)

func main() {
	MainWindow{
		AssignTo: &mw,
		Title:    "chopsuey IRC v0.2",
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

	logfilename := "./log/" + time.Now().Format("20060102150405.999999999") + ".log"
	logfile, err := os.Create(logfilename)
	checkErr(err)
	defer logfile.Close()
	l := &tsoLogger{}
	l.LogFn = func(msg string) {
		io.WriteString(logfile, msg+"\n")
	}
	logging.SetLogger(l)

	tabWidget.CurrentIndexChanged().Attach(func() {
		currentTab := getCurrentTab()
		currentTab.SetTitle(strings.TrimPrefix(currentTab.Title(), "* "))
		children := currentTab.Children()
		for i := 0; i < children.Len(); i++ {
			child := children.At(i)
			typeStr := reflect.TypeOf(child).String()
			if typeStr == "*main.MyLineEdit" {
				lineEdit := child.(*MyLineEdit)
				lineEdit.SetFocus()
			}
		}
	})

	clientCfg, err = getClientConfig()
	if err != nil {
		log.Println("error parsing config.json", err)
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
		statusBar.SetText("error parsing config.json")
	} else {
		for _, cfg := range clientCfg.AutoConnect {
			statusBar.SetText("connecting to " + cfg.ServerString() + "...")
			servConn := newServerConnection(cfg)
			servConn.connect()
		}
	}

	mw.Run()
}

func getCurrentTab() *walk.TabPage {
	return tabWidget.Pages().At(tabWidget.CurrentIndex())
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
