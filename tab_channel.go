package main

import (
	"math/rand"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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
			statusBar.SetText(t.statusText)
		}
	})

	t.statusText = servState.tab.statusText
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
		t.nickListBoxModel.Items = nicks
		t.nickListBoxModel.PublishItemsReset()
		t.nickListToggle.SetText(text)
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
	})
}

func NewChannelTab(servConn *serverConnection, servState *serverState, chanState *channelState) *tabChannel {
	t := &tabChannel{}
	t.nickColors = map[string]int{}
	clientState.AppendTab(t)
	t.tabTitle = chanState.channel

	chanState.nickList = newNickList()
	t.nickListToggle = &walk.PushButton{}
	t.nickListBox = &walk.ListBox{}
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
				Spacing:     2,
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
							} else {
								t.nickListHidden = true
								t.nickListBoxSize = nlSize
								nlSize.Width = 0
								tbSize.Width += t.nickListBoxSize.Width
							}

							t.nickListBox.SetSize(nlSize)
							t.textBuffer.SetSize(tbSize)
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

		t.textBuffer.KeyPress().Attach(ctrlTab)
		t.textInput = NewTextInput(t, &commandContext{
			servConn:  servConn,
			tab:       t,
			servState: servState,
			chanState: chanState,
			pmState:   nil,
		})
		checkErr(t.tabPage.Children().Add(t.textInput))

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
