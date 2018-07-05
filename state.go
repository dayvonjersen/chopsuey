package main

/*
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
*/

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
	tab         *tabViewServer
}

type channelState struct {
	channel string
	topic   string
	nicks   *nickList
	tab     *tabViewChannel
}

type privmsgState struct {
	nick string
	tab  *tabViewPrivmsg
}
