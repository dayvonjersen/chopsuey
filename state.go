// +build ignore

package main

import "github.com/lxn/walk"

var (
	servers  []*serverState
	tabPages []*walk.TabPage
	tabs []*abstractTab
)

func example() {
	serverState.networkName = "ROXNet"
	serverState.tab.update(serverState)

	serverConn.Join("#go-nuts")
	serverState.channels["#go-nuts"] = &channelState{
		"#go-nuts",
		"",
		newNickList(),
		createChannelTab(),
	}

	on(433, func(ch, topic string) {
		if channelSt, ok := serverState[ch]; ok {
			channelSt.topic = topic
			channelSt.tab.update(serverState, channelSt)
		} else {
			error("got topic but user not on channel")
		}
	})
}

func moreexample() {
	onIndexChanged(index) {
		for _, t := range tabs {
			if t.Id() == index {
				statusBar.SetText(t.StatusText())
				break
			}
		}
	}
}
		

///////////////////////////////////////////////////////////
// State
///////////////////////////////////////////////////////////

type userState struct {
	nick string
	// other stuff like OPER...
}

type serverState struct {
	connected   bool
	hostname    string
	port        int
	ssl         bool
	networkName string
	user        *userState
	channels    map[string]*channelState
	privmsgs    map[string]*privmsgState
	tab         *serverTab
}

type channelState struct {
	channel string
	topic   string
	nicks   *nickList
	tab     *channelTab
}

type privmsgState struct {
	nick string
	tab  *privmsgTab
}

///////////////////////////////////////////////////////////
// UI
///////////////////////////////////////////////////////////

type baseTab struct {
	tabPage *walk.TabPage
	index   int
	statusText string
}

func (b *baseTab) Id() int {
	return index
}

func (b *baseTab) StatusText() string {
	return statusText
}

type abstractTab interface {
	id()
	statusText()
	focus()
	close()
}

type chatboxTab struct {
	baseTab
	title         string
	textBuffer    *walk.TextEdit
	textInput     *MyLineEdit
	msgHistory    []string
	MsgHistoryIdx int
	tabComplete   *tabComplete
}

// this gets inherited by subtypes
func (cb *chatboxTab) printMessage(msg string) {
	// ...
}

type serverTab struct {
	chatboxTab
}

type channelTab struct {
	chatboxTab
	nickListBox      *walk.ListBox
	nickListBoxModel *listBoxModel
	topicInput       *walk.LineEdit
}

// this is specific only to this type
func (ch *channelTab) updateNickList() {
	// ...
}

func (ch *channelTab) update(s *serverState, c *channelState) {
	ch.statusText = s.user.Nick + " connected to " + s.networkName
}

type privmsgTab struct {
	chatboxTab
}

type channelListTab struct {
	baseTab
	// channel list does not inherit from chatbox
}
