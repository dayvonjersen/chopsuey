package main

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	var (
		mw          *walk.MainWindow
		nickListBox *walk.ListBox
		textBuffer  *walk.TextEdit
		textInput   *walk.LineEdit
	)
	nickListBoxModel := &listboxModel{}

	go func() {
		for i := 0; i < 100; i++ {
			nickListBoxModel.Items = append(nickListBoxModel.Items, fmt.Sprintf("shazbot%03d", i))
			nickListBoxModel.PublishItemChanged(i)
			<-time.After(time.Second)
		}
	}()

	MainWindow{
		AssignTo: &mw,
		Title:    "IRC",
		MinSize:  Size{480, 640},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			HSplitter{
				AlwaysConsumeSpace: true,
				Children: []Widget{
					TextEdit{
						MinSize:    Size{380, 640},
						AssignTo:   &textBuffer,
						ReadOnly:   true,
						Persistent: true,
					},
					ListBox{
						MinSize:    Size{100, 640},
						AssignTo:   &nickListBox,
						Model:      nickListBoxModel,
						Persistent: true,
					},
				},
			},
			LineEdit{
				AssignTo: &textInput,
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						textBuffer.AppendText("<nobody> " + textInput.Text() + "\r\n")
						textInput.SetText("")
					}
				},
			},
		},
	}.Create()

	go func() {
		for {
			textBuffer.AppendText("<nobody> lel\r\n")
			<-time.After(time.Second * 2)
		}
	}()

	mw.Run()
}

type listboxModel struct {
	walk.ListModelBase
	Items []string
}

func (m *listboxModel) ItemCount() int {
	return len(m.Items)
}

func (m *listboxModel) Value(index int) interface{} {
	return m.Items[index]
}
