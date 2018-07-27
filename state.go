package main

var clientState *_clientState // global instance
type _clientState struct {
	cfg *clientConfig
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
		servState.channels[channel] = chanState

		tabMan.Create(&tabContext{servConn: servConn, servState: servState, chanState: chanState}, index)
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
		for _, pm := range servState.privmsgs {
			i := pm.tab.Index()
			if i > index {
				index = i
			}
		}
		index++
		servState.privmsgs[nick] = pmState
		tabMan.Create(&tabContext{servConn: servConn, servState: servState, pmState: pmState}, index)
	}
	return pmState
}

// ... is ensureServerState() our solution??? :o
