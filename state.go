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
