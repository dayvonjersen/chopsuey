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

	nickListHidden bool
	size, size2    walk.Size
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
	text := strconv.Itoa(count) + " user"
	if count != 1 {
		text += "s"
	}
	if ops > 0 {
		text += ", " + strconv.Itoa(ops) + " op"
		if ops != 1 {
			text += "s"
		}
	}

	mw.WindowBase.Synchronize(func() {
		t.nickListBoxModel.Items = nicks
		t.nickListBoxModel.PublishItemsReset()
		t.nickListToggle.SetText(text)
	})
}

func (t *tabChannel) Resize() {
	mw.WindowBase.Synchronize(func() {

		s := t.nickListBox.Size()
		s2 := t.textBuffer.Size()

		if s.Width != 0 {
			t.size = s
		}
		if t.nickListHidden {
			s2.Width += s.Width
			s.Width = 0
		}

		t.nickListBox.SetSize(s)
		t.textBuffer.SetSize(s2)
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

	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(t.tabPage)

		t.size = walk.Size{}
		t.size2 = walk.Size{}
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
					Text:     "OK",
					OnClicked: func() {
						mw.WindowBase.Synchronize(func() {
							s := t.nickListBox.Size()
							s2 := t.textBuffer.Size()
							if t.nickListHidden {
								t.nickListHidden = false
								s.Width = t.size.Width
								t.size2 = s2
								s2.Width -= s.Width
							} else {
								t.nickListHidden = true
								t.size = s
								s.Width = 0
								s2.Width += t.size.Width
							}
							t.textBuffer.SetSize(s2)
							t.nickListBox.SetSize(s)
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

						pmState, ok := servState.privmsgs[nick.name]
						if !ok {
							pmState = &privmsgState{
								nick: nick.name,
							}
							pmState.tab = NewPrivmsgTab(servConn, servState, pmState)
						}
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
