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
