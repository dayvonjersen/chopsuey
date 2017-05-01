package main

import (
	"log"
	"sort"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type channelList struct {
	mu      *sync.Mutex
	mdl     *channelListModel
	tabPage *walk.TabPage
}

func (cl *channelList) Add(channel string, users int, topic string) {
	item := &channelListItem{
		channel: channel,
		users:   users,
		topic:   topic,
	}
	cl.mdl.items = append(cl.mdl.items, item)
	cl.mdl.PublishRowsReset()
	cl.mdl.Sort(cl.mdl.sortColumn, cl.mdl.sortOrder)
}

func newChannelList(servConn *serverConnection) *channelList {
	cl := &channelList{
		mu:  &sync.Mutex{},
		mdl: new(channelListModel),
	}

	var tbl *walk.TableView

	mw.WindowBase.Synchronize(func() {
		var err error
		cl.tabPage, err = walk.NewTabPage()
		checkErr(err)
		cl.tabPage.SetTitle("channels")
		cl.tabPage.SetLayout(walk.NewVBoxLayout())
		builder := NewBuilder(cl.tabPage)

		w := float64(mw.ClientBounds().Width)

		TableView{
			AssignTo:         &tbl,
			Model:            cl.mdl,
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
				channel := cl.mdl.items[tbl.CurrentIndex()].channel
				servConn.join(channel)
			},
		}.Create(builder)
		PushButton{
			Text: "Close Tab",
			OnClicked: func() {
				mw.WindowBase.Synchronize(func() {
					checkErr(tabWidget.Pages().Remove(cl.tabPage))
					checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Len() - 1))
					tabWidget.SaveState()
				})
			},
		}.Create(builder)
		checkErr(tabWidget.Pages().Add(cl.tabPage))
		checkErr(tabWidget.SetCurrentIndex(tabWidget.Pages().Index(cl.tabPage)))
		tabWidget.SaveState()
	})

	return cl
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
			return cmp(a.users < b.users)
		case 2:
			return cmp(a.topic < b.topic)
		}

		log.Panicln("unexpected column:", m.sortColumn)
		return false
	})

	return m.SorterBase.Sort(col, order)
}
