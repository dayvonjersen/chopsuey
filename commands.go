package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type commandContext struct {
	servConn *serverConnection
	tab      tabWithInput

	servState *serverState
	chanState *channelState
	pmState   *privmsgState
}

type clientCommand func(ctx *commandContext, args ...string)

var (
	clientCommands map[string]clientCommand
	scriptAliases  = map[string]string{}
)

func init() {
	clientCommands = map[string]clientCommand{
		// connectivity
		"connect":    connectCmd,
		"disconnect": disconnectCmd,
		"quit":       quitCmd,
		"reconnect":  reconnectCmd,
		"server":     serverCmd,

		// other core functionality
		"clear":  clearCmd,
		"close":  closeCmd,
		"ctcp":   ctcpCmd,
		"join":   joinCmd,
		"kick":   kickCmd,
		"list":   listCmd,
		"me":     meCmd,
		"mode":   modeCmd,
		"msg":    privmsgCmd,
		"nick":   nickCmd,
		"notice": noticeCmd,
		"part":   partCmd,
		"rejoin": rejoinCmd,
		"topic":  topicCmd,

		// stubs
		"help": helpCmd,
		"exit": exitCmd,

		// experimental/WIP
		"send": sendCmd,

		// scripting
		"script":     scriptCmd,
		"register":   registerCmd,
		"unregister": unregisterCmd,

		// debugging
		"raw": rawCmd,
	}
}

func connectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case CONNECTION_EMPTY:
		clientError(ctx.tab, "no network specified (use /server)")

	case CONNECTING, CONNECTION_START:
		clientError(ctx.tab, "connection in progress: "+serverAddr(ctx.servState.hostname, ctx.servState.port))

	case CONNECTED:
		clientError(ctx.tab, "already connected to: "+serverAddr(ctx.servState.hostname, ctx.servState.port))

	case DISCONNECTED, CONNECTION_ERROR:
		ctx.servConn.Connect(ctx.servState)
	}
}

func disconnectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case DISCONNECTED, CONNECTION_ERROR, CONNECTION_EMPTY:
		clientError(ctx.tab, "already disconnected.")
		return

	case CONNECTING, CONNECTION_START, CONNECTED:
		ctx.servConn.retryConnectEnabled = false
		select {
		case <-ctx.servConn.cancelRetryConnect:
		default:
			close(ctx.servConn.cancelRetryConnect)
		}
		ctx.servConn.conn.Quit(strings.Join(args, " "))
	}
}

func quitCmd(ctx *commandContext, args ...string) {
	disconnectCmd(ctx, args...)
	for k, chanState := range ctx.servState.channels {
		chanState.tab.Close()
		delete(ctx.servState.channels, k)
	}
	for k, pmState := range ctx.servState.privmsgs {
		pmState.tab.Close()
		delete(ctx.servState.privmsgs, k)
	}
	if ctx.servState.channelList != nil {
		ctx.servState.channelList.Close()
		ctx.servState.channelList = nil
	}
	if clientState.NumTabs() > 1 {
		ctx.servState.tab.Close()
	}
}

func reconnectCmd(ctx *commandContext, args ...string) {
	clientMessage(ctx.tab, `Sorry, /reconnect doesn't work right now, please do:
/disconnect
/connect`)
	return
	/*
		// FIXME(tso): either find out why goirc panics when we do this or replace goirc altogether
			switch ctx.servState.connState {
			case CONNECTION_EMPTY:
				clientMessage(ctx.tab, "ERROR: no network specified (use /server)")
			case CONNECTING, CONNECTION_START:
				clientMessage(ctx.tab, "ERROR: connection in progress: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))

			case CONNECTED:
				disconnectCmd(ctx, args...)
				<-time.After(time.Second * 3)
				fallthrough
			case DISCONNECTED, CONNECTION_ERROR:
				ctx.servConn.retryConnectEnabled = true
				connectCmd(ctx, args...)
			}
	*/
}

func serverCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		clientMessage(ctx.tab, "usage: /server [host] [port (default 6667)]\r\n  ssl: /server [host] +[port (default 6697)]")
		return
	}

	// FIXME(tso): abstract opening a new server connection/tab by reusing this code
	hostname := args[0]
	port := 6667
	ssl := false
	if len(args) > 1 {
		portStr := args[1]
		if portStr[0] == '+' {
			ssl = true
			port = 6697
			portStr = portStr[1:]
		}
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	servState := &serverState{}
	// FIXME(tso): empty tab is a nightmare holy fuck
	if clientState.NumTabs() == 1 && ctx.servState != nil && ctx.servState.tab != nil && ctx.servState.connState == CONNECTION_EMPTY {
		servState = ctx.servState
	}

	servState.connState = CONNECTION_EMPTY
	servState.hostname = hostname
	servState.port = port
	servState.ssl = ssl
	servState.networkName = fmt.Sprintf("%s:%d", hostname, port)
	servState.user = &userState{
		nick: ctx.servState.user.nick,
	}
	servState.channels = map[string]*channelState{}
	servState.privmsgs = map[string]*privmsgState{}

	servConn := NewServerConnection(servState, func() {})

	// FIXME(tso): empty tab is a nightmare holy fuck
	if !(clientState.NumTabs() == 1 && ctx.servState != nil && ctx.servState.tab != nil && ctx.servState.connState == CONNECTION_EMPTY) {
		servView := NewServerTab(servConn, servState)
		servState.tab = servView
	}

	servConn.Connect(servState)
}

func clearCmd(ctx *commandContext, args ...string) {
	ctx.tab.Clear()
}

func closeCmd(ctx *commandContext, args ...string) {
	if ctx.tab == ctx.servState.tab {
		quitCmd(ctx, args...)

		if clientState.NumTabs() == 1 {
			ctx.servState = &serverState{
				connState: CONNECTION_EMPTY,
				channels:  map[string]*channelState{},
				privmsgs:  map[string]*privmsgState{},
				user:      ctx.servState.user,
				tab:       ctx.servState.tab,
			}
			ctx.servState.tab.Update(ctx.servState)
		} else {
			ctx.servState.tab.Close()
		}
		return
	}
	if ctx.chanState != nil {
		partCmd(ctx, args...)
		delete(ctx.servState.channels, ctx.chanState.channel)
	} else if ctx.pmState != nil {
		delete(ctx.servState.privmsgs, ctx.pmState.nick)
	}
	ctx.tab.Close()
}

func ctcpCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		clientMessage(ctx.tab, "usage: /ctcp [nick] [message] [args...]")
		return
	}
	ctx.servConn.conn.Ctcp(args[0], args[1], args[2:]...)
}

func joinCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		clientMessage(ctx.tab, "usage: /join [#channel]")
		return
	}
	ctx.servConn.conn.Join(args[0])
}

func kickCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		clientMessage(ctx.tab, "usage: /kick [nick] [(optional) reason...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Kick(ctx.chanState.channel, args[0], args[1:]...)
	} else {
		clientError(ctx.tab, "/kick only works for channels.")
	}
}

func listCmd(ctx *commandContext, args ...string) {
	if ctx.servState.connState != CONNECTED {
		clientError(ctx.tab, "Can't LIST: not connected to any network")
		return
	}
	if ctx.servState.channelList != nil {
		if ctx.servState.channelList.complete {
			clientMessage(ctx.tab, "refreshing LIST...")
			ctx.servState.channelList.Clear()
		}
		if ctx.servState.channelList.inProgress {
			clientMessage(ctx.tab, ctx.servState.networkName, "is still sending results!")
			return
		}
	}
	ctx.servConn.conn.Raw("LIST")
}

func meCmd(ctx *commandContext, args ...string) {
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		clientMessage(ctx.tab, "usage: /me [message...]")
		return
	}
	var dest string
	if ctx.chanState != nil {
		dest = ctx.chanState.channel
	} else if ctx.pmState != nil {
		dest = ctx.pmState.nick
	} else {
		clientError(ctx.tab, "ERROR: /me can only be used in channels and private messages")
		return
	}
	ctx.servConn.conn.Action(dest, msg)
	actionMessage(ctx.tab, ctx.servState.user.nick, msg)
}

func modeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		clientMessage(ctx.tab, "usage: /mode [#channel or your nick] [mode] [nicks...]")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func privmsgCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 || isChannel(args[0]) {
		clientMessage(ctx.tab, "usage: /msg [nick] [message...]")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	// FIXME(tso): always inline messages to services
	//             probably should add an option to disable it too

	if isService(nick) {
		ctx.servConn.conn.Privmsg(nick, msg)
		noticeMessage(
			getCurrentTabForServer(ctx.servState),
			ctx.servState.user.nick, nick, msg)
		return
	}

	pmState := ensurePmState(ctx.servConn, ctx.servState, nick)

	ctx.servConn.conn.Privmsg(nick, msg)
	privateMessage(pmState.tab, ctx.servState.user.nick, msg)
	mw.WindowBase.Synchronize(func() {
		checkErr(tabWidget.SetCurrentIndex(pmState.tab.Index()))
	})
}

func nickCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		clientMessage(ctx.tab, "usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func noticeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		clientMessage(ctx.tab, "usage: /notice [#channel or nick] [message...]")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	noticeMessage(ctx.tab, ctx.servState.user.nick, args[0], msg)
}

func partCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, strings.Join(args, " "), ctx.servState)
	} else {
		clientError(ctx.tab, "ERROR: /part only works for channels. Try /close")
	}
}

func rejoinCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, "rejoining...", ctx.servState)
		ctx.servConn.conn.Join(ctx.chanState.channel)
	} else {
		clientError(ctx.tab, "ERROR: /rejoin only works for channels.")
	}
}

func topicCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		clientMessage(ctx.tab, "usage: /topic [new topic...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Topic(ctx.chanState.channel, args...)
	} else {
		clientError(ctx.tab, "ERROR: /topic can only be used in channels")
	}
}

func helpCmd(ctx *commandContext, args ...string) {
	// TODO(tso): print usage for individual commands
	clientMessage(ctx.tab,
		`connect to a network:
/server irc.example.org

to quit the application:
/exit

disconnect from a network without closing all your tabs:
/disconnect

disconnect from a network AND close all associated tabs:
/quit
or on the server connection tab:
/close

leave a channel without closing tab:
/part

leave a channel or privmsg and close tab:
/close
`)
}

func exitCmd(ctx *commandContext, args ...string) {
	// FIXME(tso): cleaner shutdown
	os.Exit(0)
}

func sendCmd(ctx *commandContext, args ...string) {
	who := args[0]
	file := "rfc2812.txt"
	fileTransfer(ctx.servConn, who, file)
}

func scriptCmd(ctx *commandContext, args ...string) {
	if len(args) < 0 {
		clientMessage(ctx.tab, "usage: /script [file in "+SCRIPTS_DIR+"] [args...]")
		return
	}

	if ctx.chanState == nil && ctx.pmState == nil {
		clientMessage(ctx.tab, "ERROR: scripts only work in channels and private messages!")
		return
	}

	scriptFile := filepath.Base(args[0])

	f, err := os.Open(SCRIPTS_DIR + scriptFile)
	checkErr(err)
	if os.IsNotExist(err) {
		clientError(ctx.tab, "script not found: "+scriptFile)
		return
	}
	{
		finfo, err := f.Stat()
		checkErr(err)
		if finfo.IsDir() {
			clientError(ctx.tab, "script not found: "+scriptFile)
			return
		}
	}
	f.Close()

	args = append([]string{SCRIPTS_DIR + scriptFile}, args[1:]...)

	var bin string
	ext := filepath.Ext(scriptFile)
	switch ext {
	case ".go":
		bin = "go"
		args = append([]string{"run"}, args...)
	case ".php":
		bin = "php"
	case ".pl":
		bin = "perl"
	case ".py":
		bin = "python"
	case ".rb":
		bin = "ruby"
	case ".sh":
		bin = "bash"
	default:
		clientError(ctx.tab, "unsupported script type: "+ext)
		return
	}

	go func() {
		cmd := exec.Command(bin, args...)
		stdout := bytes.NewBuffer([]byte{})
		stderr := bytes.NewBuffer([]byte{})
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			clientError(ctx.tab, append([]string{bin}, args...)...)
			clientError(ctx.tab, "returned:", err.Error())
		}
		out := strings.TrimSpace(stdout.String())
		err := strings.TrimSpace(stderr.String())
		if err != "" {
			clientError(ctx.tab, append([]string{bin}, args...)...)
			clientError(ctx.tab, "returned:", err)
		}
		if out != "" {
			ctx.tab.Send(out)
		}
	}()
}

func registerCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		clientMessage(ctx.tab, "usage: /register [alias] [script file]")
		return
	}

	name := args[0]
	file := args[1]

	if _, ok := clientCommands[name]; ok {
		if alias, ok := scriptAliases[name]; ok {
			clientMessage(ctx.tab, "overwriting previous alias of "+alias+" for /"+name)
		} else {
			clientError(ctx.tab, "cannot overwrite built-in client command /"+name)
			return
		}
	}

	scriptAliases[name] = file
	clientCommands[name] = func(ctx *commandContext, args ...string) {
		scriptCmd(ctx, append([]string{file}, args...)...)
	}
	clientMessage(ctx.tab, "/"+name+" registered as alias to "+file)
}

func unregisterCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		clientMessage(ctx.tab, "usage: /unregister [alias]")
		return
	}
	name := args[0]
	alias, ok := scriptAliases[name]
	if !ok {
		clientError(ctx.tab, "/"+name+" is not a registered alias of any script")
		return
	}
	delete(scriptAliases, name)
	delete(clientCommands, name)
	clientMessage(ctx.tab, "/"+name+" ("+alias+") unregistered")
}

func rawCmd(ctx *commandContext, args ...string) {
	ctx.servConn.conn.Raw(strings.Join(args, " "))
}
