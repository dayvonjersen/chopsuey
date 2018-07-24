package main

import (
	"math/rand"
	"strconv"
	"syscall"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type listBoxModel struct {
	walk.ListModelBase
	Items []string
}

func (m *listBoxModel) ItemCount() int {
	return len(m.Items)
}

func (m *listBoxModel) Value(index int) interface{} {
	return m.Items[index]
}

type tabChannel struct {
	tabChatbox
	topicInput       *walk.LineEdit
	nickListToggle   *walk.PushButton
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	send             func(string)

	nickColors map[string]int

	nickListHidden  bool
	nickListBoxSize walk.Size
}

func (t *tabChannel) NickColor(nick string) int {
	if color, ok := t.nickColors[nick]; ok {
		return color
	}
	r := rand.Intn(98)
	for !colorVisible(colorPalette[r], globalBackgroundColor) {
		r = rand.Intn(98)
	}
	t.nickColors[nick] = r
	return t.nickColors[nick]
}

func (t *tabChannel) Send(message string) {
	t.send(message)
}

func (t *tabChannel) Update(servState *serverState, chanState *channelState) {
	t.disconnected = servState.connState != CONNECTED
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
		t.topicInput.SetText(chanState.topic)
		if t.HasFocus() {
			SetStatusBarIcon(t.statusIcon)
			SetStatusBarText(t.statusText)
		}
	})

	t.statusIcon = servState.tab.statusIcon
	t.statusText = servState.tab.statusText

	SetSystrayContextMenu()
}

func (t *tabChannel) updateNickList(chanState *channelState) {
	nicks := chanState.nickList.StringSlice()
	count := len(nicks)
	ops := 0
	for _, n := range nicks {
		m := nickRegex.FindAllStringSubmatch(n, 01)
		if m[0][1] != "" && m[0][1] != "+" {
			ops++
		}
	}
	text := strconv.Itoa(count) + pluralize(" user", count)
	if ops > 0 {
		text += ", " + strconv.Itoa(ops) + pluralize(" op", ops)
	}

	mw.WindowBase.Synchronize(func() {
		bg := globalBackgroundColor
		r, g, b := byte((bg>>16)&0xff), byte((bg>>8)&0xff), byte(bg&0xff)
		brush, err := walk.NewSolidColorBrush(walk.RGB(r, g, b))
		checkErr(err)
		t.nickListBox.SetBackground(brush)
		t.nickListBoxModel.Items = nicks
		t.nickListBoxModel.PublishItemsReset()
		t.nickListToggle.SetText(text)
		ShowScrollBar(t.nickListBox.Handle(), win.SB_HORZ, 0)
	})
}

func (t *tabChannel) Resize() {
	mw.WindowBase.Synchronize(func() {
		nlSize := t.nickListBox.Size()
		tbSize := t.textBuffer.Size()

		if nlSize.Width != 0 {
			t.nickListBoxSize = nlSize
		}

		if t.nickListHidden {
			tbSize.Width += nlSize.Width
			nlSize.Width = 0
		}

		t.nickListBox.SetSize(nlSize)
		t.textBuffer.SetSize(tbSize)

		ShowScrollBar(t.nickListBox.Handle(), win.SB_HORZ, 0)
	})
}

func NewChannelTab(servConn *serverConnection, servState *serverState, chanState *channelState) *tabChannel {
	t := &tabChannel{}
	t.nickColors = map[string]int{}
	clientState.AppendTab(t)
	t.tabTitle = chanState.channel

	chanState.nickList = newNickList()
	t.nickListToggle = &walk.PushButton{}
	t.nickListBoxModel = &listBoxModel{}

	t.topicInput = &walk.LineEdit{}

	t.send = func(msg string) {
		servConn.conn.Privmsg(chanState.channel, msg)
		nick := chanState.nickList.Get(servState.user.nick)
		privateMessage(t, nick.String(), msg)
	}

	t.chatlogger = NewChatLogger(servState.networkName + "-" + chanState.channel)

	t.nickListHidden = false
	t.nickListBoxSize = walk.Size{}

	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)

		Composite{
			Layout: HBox{
				MarginsZero: true,
				SpacingZero: true,
			},
			Children: []Widget{
				LineEdit{
					AssignTo: &t.topicInput,
					ReadOnly: true,
				},
				PushButton{
					AssignTo: &t.nickListToggle,
					Text:     "0 users",
					OnClicked: func() {
						mw.WindowBase.Synchronize(func() {
							nlSize := t.nickListBox.Size()
							tbSize := t.textBuffer.Size()

							if t.nickListHidden {
								t.nickListHidden = false
								nlSize.Width = t.nickListBoxSize.Width
								tbSize.Width -= nlSize.Width
								t.nickListBox.SetVisible(true)
							} else {
								t.nickListHidden = true
								t.nickListBoxSize = nlSize
								nlSize.Width = 0
								tbSize.Width += t.nickListBoxSize.Width
								t.nickListBox.SetVisible(false)
							}

							t.nickListBox.SetSize(nlSize)
							t.textBuffer.SetSize(tbSize)
							ShowScrollBar(t.nickListBox.Handle(), win.SB_HORZ, 0)
						})
					},
				},
			},
		}.Create(builder)

		t.textBuffer = &RichEdit{}

		HSplitter{
			AlwaysConsumeSpace: true,
			HandleWidth:        2,
			Children: []Widget{
				RichEditDecl{
					AssignTo:      &t.textBuffer,
					StretchFactor: 3,
				},
				ListBox{
					AssignTo:           &t.nickListBox,
					Model:              t.nickListBoxModel,
					AlwaysConsumeSpace: false,
					StretchFactor:      1,
					OnItemActivated: func() {
						nick := newNick(t.nickListBoxModel.Items[t.nickListBox.CurrentIndex()])

						pmState := ensurePmState(servConn, servState, nick.name)
						mw.WindowBase.Synchronize(func() {
							checkErr(tabWidget.SetCurrentIndex(pmState.tab.Index()))
						})
					},
				},
			},
		}.Create(builder)
		ShowScrollBar(t.nickListBox.Handle(), win.SB_HORZ, 0)

		t.textBuffer.KeyPress().Attach(ctrlTab)
		t.textInput = NewTextInput(t, &commandContext{
			servConn:  servConn,
			tab:       t,
			servState: servState,
			chanState: chanState,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))

		// remove borders
		win.SetWindowLong(t.topicInput.Handle(), win.GWL_EXSTYLE, 0)
		win.SetWindowLong(t.textInput.Handle(), win.GWL_EXSTYLE, 0)
		win.SetWindowLong(t.nickListBox.Handle(), win.GWL_STYLE, win.WS_TABSTOP|win.WS_VISIBLE|win.LBS_NOINTEGRALHEIGHT|win.LBS_NOTIFY)

		// override WndProc just to set bg/fg colors fml
		var origWndProcPtr uintptr
		wndProc := func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
			if msg == win.WM_CTLCOLORLISTBOX {
				hdc := win.HDC(wParam)
				win.SetTextColor(hdc, rgb2COLORREF(globalForegroundColor))
				win.SetBkColor(hdc, rgb2COLORREF(globalBackgroundColor))
				return win.TRUE
			}

			return win.CallWindowProc(origWndProcPtr, hwnd, msg, wParam, lParam)
		}
		origWndProcPtr = win.SetWindowLongPtr(t.nickListBox.Parent().Handle(), win.GWLP_WNDPROC, syscall.NewCallback(wndProc))

		applyThemeToTab(t)
		{
			index := servState.tab.Index()
			if servState.channelList != nil {
				index = servState.channelList.Index()
			}
			for _, ch := range servState.channels {
				i := ch.tab.Index()
				if i > index {
					index = i
				}
			}
			index++

			checkErr(tabWidget.Pages().Insert(index, t.tabPage))
		}
		index := tabWidget.Pages().Index(t.tabPage)
		checkErr(tabWidget.SetCurrentIndex(index))
		tabWidget.SaveState()
		t.Focus()
	})
	chanState.tab = t
	servState.channels[chanState.channel] = chanState
	servState.tab.Update(servState)
	return t
}
