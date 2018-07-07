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
	tab         *tabViewServer
	channelList *tabViewChannelList
}

func (servState *serverState) IndexMax() int {
	max := servState.tab.Index()
	if servState.channelList != nil {
		index := servState.channelList.Index()
		if index > max {
			max = index
		}
	}
	for _, chanState := range servState.channels {
		index := chanState.tab.Index()
		if index > max {
			max = index
		}
	}
	for _, pmState := range servState.privmsgs {
		index := pmState.tab.Index()
		if index > max {
			max = index
		}
	}
	return max
}

type channelState struct {
	channel  string
	topic    string
	nickList *nickList
	tab      *tabViewChannel
}

type privmsgState struct {
	nick string
	tab  *tabViewPrivmsg
}
