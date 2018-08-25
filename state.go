package main

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
	contexts := tabMan.FindAll(allServerTabsFinder(servState))
	ret := []tabWithTextBuffer{}
	for _, ctx := range contexts {
		if t, ok := ctx.tab.(tabWithTextBuffer); ok {
			ret = append(ret, t)
		}
	}
	return ret
}

func (servState *serverState) CurrentTab() tabWithTextBuffer {
	ctx := tabMan.Find(currentServerTabFinder(servState))
	return ctx.tab.(tabWithTextBuffer)
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

		// TODO(tso): make a finderFunc instead
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

		ctx := tabMan.Create(&tabContext{servConn: servConn, servState: servState, chanState: chanState}, index)
		tab := newChannelTab(servConn, servState, chanState, index)
		ctx.tab = tab
		chanState.tab = tab
	}
	return chanState
}

func ensurePmState(servConn *serverConnection, servState *serverState, nick string) *privmsgState {
	pmState, ok := servState.privmsgs[nick]
	if !ok {
		pmState = &privmsgState{
			nick: nick,
		}

		// TODO(tso): make a finderFunc instead
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

		ctx := tabMan.Create(&tabContext{servConn: servConn, servState: servState, pmState: pmState}, index)
		tab := newPrivmsgTab(servConn, servState, pmState, index)
		ctx.tab = tab
		pmState.tab = tab
	}
	return pmState
}
