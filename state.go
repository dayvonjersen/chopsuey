package main

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
