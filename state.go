package main

var clientState *_clientState // global instance
type _clientState struct {
	cfg *clientConfig

	connections []*serverConnection
	servers     []*serverState
	tabs        []tab
}

func (clientState *_clientState) AppendTab(t tab) {
	clientState.tabs = append(clientState.tabs, t)
}

func (clientState *_clientState) RemoveTab(t tab) {
	index := t.Index()
	for i, tab := range clientState.tabs {
		if tab.Index() == index {
			clientState.tabs = append(clientState.tabs[0:i], clientState.tabs[i+1:]...)
			break
		}
	}
}

func (clientState *_clientState) NumTabs() int {
	return len(clientState.tabs)
}

func (clientState *_clientState) CurrentTab() tab {
	index := tabWidget.CurrentIndex()
	for _, t := range clientState.tabs {
		if t.Index() == index {
			return t
		}
	}
	return nil
}

type userState struct {
	nick string
	// other stuff like OPER...
}

const (
	CONNECTION_EMPTY = iota
	DISCONNECTED
	CONNECTING
	CONNECTION_ERROR
	CONNECTION_START
	CONNECTED
)

type serverState struct {
	connState   int
	lastError   error
	hostname    string
	port        int
	ssl         bool
	networkName string
	user        *userState
	channels    map[string]*channelState
	privmsgs    map[string]*privmsgState
	tab         *tabServer
	channelList *tabChannelList
}

func (servState *serverState) AllTabs() []tabWithTextBuffer {
	ret := []tabWithTextBuffer{servState.tab}
	for _, chanState := range servState.channels {
		ret = append(ret, chanState.tab)
	}
	for _, pmState := range servState.privmsgs {
		ret = append(ret, pmState.tab)
	}
	return ret
}

func (servState *serverState) CurrentTab() tabWithTextBuffer {
	index := tabWidget.CurrentIndex()
	if servState.tab.Index() == index {
		return servState.tab
	}
	for _, ch := range servState.channels {
		if ch.tab.Index() == index {
			return ch.tab
		}
	}
	for _, pm := range servState.privmsgs {
		if pm.tab.Index() == index {
			return pm.tab
		}
	}
	return servState.tab
}

type channelState struct {
	channel  string
	topic    string
	nickList *nickList
	tab      *tabChannel
}

type privmsgState struct {
	nick string
	tab  *tabPrivmsg
}

func ensureChanState(servConn *serverConnection, servState *serverState, channel string) *channelState {
	chanState, ok := servState.channels[channel]
	if !ok {
		chanState = &channelState{
			channel:  channel,
			nickList: newNickList(),
		}
		chanState.tab = NewChannelTab(servConn, servState, chanState)
		servState.channels[channel] = chanState
	}
	return chanState
}

// ok so maybe generics might be useful sometimes
func ensurePmState(servConn *serverConnection, servState *serverState, nick string) *privmsgState {
	pmState, ok := servState.privmsgs[nick]
	if !ok {
		pmState = &privmsgState{
			nick: nick,
		}
		pmState.tab = NewPrivmsgTab(servConn, servState, pmState)
		servState.privmsgs[nick] = pmState
	}
	return pmState
}

// ... is ensureServerState() our solution??? :o
