package main

import (
	"log"
	"sort"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type tabChannelList struct {
	tabCommon
	mu  *sync.Mutex
	mdl *channelListModel

	complete, inProgress bool
}

func (t *tabChannelList) Add(channel string, users int, topic string) {
	item := &channelListItem{
		channel: channel,
		users:   users,
		topic:   topic,
	}
	t.mdl.items = append(t.mdl.items, item)
	if t.complete || len(t.mdl.items)%50 == 0 {
		// t.tabPage.SetSuspended(true)
		// defer t.tabPage.SetSuspended(false)
		mw.WindowBase.Synchronize(func() {
			t.mdl.PublishRowsReset()
			t.mdl.Sort(t.mdl.sortColumn, t.mdl.sortOrder)
		})
	}
}

func (t *tabChannelList) Clear() {
	t.mdl.items = []*channelListItem{}
	t.tabPage.SetSuspended(true)
	defer t.tabPage.SetSuspended(false)
	t.mdl.PublishRowsReset()
	t.mdl.Sort(t.mdl.sortColumn, t.mdl.sortOrder)
}

func (t *tabChannelList) Title() string {
	return t.tabTitle
}

func (t *tabChannelList) Focus() {
	mw.WindowBase.Synchronize(func() {
		t.tabPage.SetTitle(t.Title())
		SetStatusBarIcon(t.statusIcon)
		SetStatusBarText(t.statusText)
	})
}

func (t *tabChannelList) Update(servState *serverState) {
	t.statusIcon = servState.tab.statusIcon
	t.statusText = servState.tab.statusText
	if t.HasFocus() {
		SetStatusBarIcon(t.statusIcon)
		SetStatusBarText(t.statusText)
	}

	if servState.connState != CONNECTED {
		t.tabTitle = "(channels)"
	} else {
		t.tabTitle = "channels"
	}
	t.tabPage.SetTitle(t.tabTitle)
}

func NewChannelList(servConn *serverConnection, servState *serverState) *tabChannelList {
	t := &tabChannelList{}
	clientState.AppendTab(t)
	t.mu = &sync.Mutex{}
	t.mdl = new(channelListModel)
	t.complete = false
	t.inProgress = false
	t.statusIcon = servState.tab.statusIcon
	t.statusText = servState.tab.statusText

	var tbl *walk.TableView

	mw.WindowBase.Synchronize(func() {
		var err error
		t.tabPage, err = walk.NewTabPage()
		checkErr(err)
		t.tabTitle = "channels"
		t.tabPage.SetTitle(t.tabTitle)
		t.tabPage.SetLayout(walk.NewVBoxLayout())

		builder := NewBuilder(t.tabPage)

		w := float64(mw.ClientBounds().Width)

		TableView{
			AssignTo:         &tbl,
			Model:            t.mdl,
			ColumnsOrderable: true,
			Columns: []TableViewColumn{
				{
					Title: "channel",
					Width: int(w * 0.2),
				},
				{
					Title: "# users",
					Width: int(w * 0.125),
				},
				{
					Title: "topic",
					Width: int(w * 0.65),
				},
			},
			OnItemActivated: func() {
				channel := t.mdl.items[tbl.CurrentIndex()].channel
				servConn.conn.Join(channel)
			},
		}.Create(builder)

		PushButton{
			Text: "Close Tab",
			OnClicked: func() {
				mw.WindowBase.Synchronize(func() {
					t.Clear()
					t.Close()
					servState.channelList = nil
				})
			},
		}.Create(builder)

		checkErr(tabWidget.Pages().Insert(servState.tab.Index()+1, t.tabPage))
		tabWidget.SetCurrentIndex(tabWidget.CurrentIndex() + 1)
		tabWidget.SaveState()
	})

	return t
}

type channelListItem struct {
	channel string
	users   int
	topic   string
}

type channelListModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*channelListItem
}

func (m *channelListModel) RowCount() int {
	return len(m.items)
}

func (m *channelListModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.channel
	case 1:
		return item.users
	case 2:
		return item.topic
	}

	log.Panicln("unexpected column:", col)
	return nil
}

func (m *channelListModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	cmp := func(x bool) bool {
		if m.sortOrder == walk.SortAscending {
			return x
		}
		return !x
	}

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]
		switch m.sortColumn {
		case 0:
			return cmp(a.channel < b.channel)
		case 1:
			if a.users == b.users {
				return cmp(a.channel < b.channel)
			}
			return cmp(a.users < b.users)
		case 2:
			if a.topic == b.topic {
				return cmp(a.channel < b.channel)
			}
			return cmp(a.topic < b.topic)
		}

		log.Panicln("unexpected column:", m.sortColumn)
		return false
	})

	return m.SorterBase.Sort(col, order)
}
