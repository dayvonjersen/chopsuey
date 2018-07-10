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
	tab      tabViewWithInput

	servState *serverState
	chanState *channelState
	pmState   *privmsgState
}

type clientCommand func(ctx *commandContext, args ...string)

var clientCommands map[string]clientCommand

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

func rawCmd(ctx *commandContext, args ...string) {
	ctx.servConn.conn.Raw(strings.Join(args, " "))
}

func helpCmd(ctx *commandContext, args ...string) {
	// TODO(tso): print usage for individual commands
	ctx.tab.Println(`TIPS AND TRICKS:

connect to a network:
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
	os.Exit(0)
}

func sendCmd(ctx *commandContext, args ...string) {
	who := args[0]
	file := "rfc2812.txt"
	fileTransfer(ctx.servConn, who, file)
}

func ctcpCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /ctcp [nick] [message] [args...]")
		return
	}
	ctx.servConn.conn.Ctcp(args[0], args[1], args[2:]...)
}

func meCmd(ctx *commandContext, args ...string) {
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		ctx.tab.Println("usage: /me [message...]")
		return
	}
	var dest string
	if ctx.chanState != nil {
		dest = ctx.chanState.channel
	} else if ctx.pmState != nil {
		dest = ctx.pmState.nick
	} else {
		ctx.tab.Println("ERROR: /me can only be used in channels and private messages")
		return
	}
	ctx.servConn.conn.Action(dest, msg)
	ctx.tab.Println(fmt.Sprintf(color("%s", LightGrey)+color(" *%s %s*", DarkGrey), now(), ctx.servState.user.nick, msg))
}

func joinCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		ctx.tab.Println("usage: /join [#channel]")
		return
	}
	ctx.servConn.conn.Join(args[0])
}

func partCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, strings.Join(args, " "), ctx.servState)
	} else {
		ctx.tab.Println("ERROR: /part only works for channels. Try /close")
	}
}

func noticeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /notice [#channel or nick] [message...]")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	ctx.tab.Println(fmt.Sprintf("%s *** %s: %s", now(), ctx.servState.user.nick, msg))
}

func privmsgCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 || args[0][0] == '#' {
		ctx.tab.Println("usage: /msg [nick] [message...]")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	pmState, ok := ctx.servState.privmsgs[nick]
	if !ok {
		pmState = &privmsgState{
			nick: nick,
		}
		pmState.tab = NewPrivmsgTab(ctx.servConn, ctx.servState, pmState)
	}

	ctx.servConn.conn.Privmsg(nick, msg)
	pmState.tab.Println(fmt.Sprintf("%s <%s> %s", now(), ctx.servState.user.nick, msg))
	mw.WindowBase.Synchronize(func() {
		checkErr(tabWidget.SetCurrentIndex(pmState.tab.Index()))
	})
}

func nickCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		ctx.tab.Println("usage: /nick [new nick]")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func connectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case CONNECTION_EMPTY:
		ctx.tab.Println("ERROR: no network specified (use /server)")
	case CONNECTING, CONNECTION_START:
		ctx.tab.Println("ERROR: connection in progress: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))
	case CONNECTED:
		ctx.tab.Println("ERROR: already connected to: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))

	case DISCONNECTED, CONNECTION_ERROR:
		ctx.servConn.Connect(ctx.servState)
	}
}

func reconnectCmd(ctx *commandContext, args ...string) {
	ctx.tab.Println(`DOESN'T WORK RIGHT NOW BUT TYPING:
/disconnect
/connect
DOES WORK

(EVEN THOUGH THAT'S LITERALLY WHAT THIS FUNCTION DID)
`)
	return
	/*
		switch ctx.servState.connState {
		case CONNECTION_EMPTY:
			ctx.tab.Println("ERROR: no network specified (use /server)")
		case CONNECTING, CONNECTION_START:
			ctx.tab.Println("ERROR: connection in progress: " + fmt.Sprintf("%s:%d", ctx.servState.hostname, ctx.servState.port))

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

func disconnectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case DISCONNECTED, CONNECTION_ERROR, CONNECTION_EMPTY:
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
	if len(tabs) > 1 {
		ctx.servState.tab.Close()
	}
}

func closeCmd(ctx *commandContext, args ...string) {
	if ctx.tab == ctx.servState.tab {
		quitCmd(ctx, args...)

		if len(tabs) == 1 {
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

func modeCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /mode [#channel or your nick] [mode] [nicks...]")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func clearCmd(ctx *commandContext, args ...string) {
	ctx.tab.Clear()
}

func topicCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /topic [new topic...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Topic(ctx.chanState.channel, args...)
	} else {
		ctx.tab.Println("ERROR: /topic can only be used in channels")
	}
}

func rejoinCmd(ctx *commandContext, args ...string) {
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, "rejoining...", ctx.servState)
		ctx.servConn.conn.Join(ctx.chanState.channel)
	} else {
		ctx.tab.Println("ERROR: /rejoin only works for channels.")
	}
}

func kickCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /kick [nick] [(optional) reason...]")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Kick(ctx.chanState.channel, args[0], args[1:]...)
	} else {
		ctx.tab.Println("ERROR: /kick only works for channels.")
	}
}

func serverCmd(ctx *commandContext, args ...string) {
	if len(args) < 1 {
		ctx.tab.Println("usage: /server [host] [port (default 6667)]\r\n  ssl: /server [host] +[port (default 6697)]")
		return
	}
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
	servState := &serverState{
		connState:   CONNECTION_EMPTY,
		hostname:    hostname,
		port:        port,
		ssl:         ssl,
		networkName: fmt.Sprintf("%s:%d", hostname, port),
		user: &userState{
			nick: ctx.servState.user.nick,
		},
		channels: map[string]*channelState{},
		privmsgs: map[string]*privmsgState{},
	}
	servConn := NewServerConnection(servState, func() {})
	if len(tabs) == 1 && ctx.servState != nil && ctx.servState.tab != nil && ctx.servState.connState == CONNECTION_EMPTY {
		servState.tab = ctx.servState.tab
	} else {
		servView := NewServerTab(servConn, servState)
		servState.tab = servView
	}
	servConn.Connect(servState)
}

func listCmd(ctx *commandContext, args ...string) {
	if ctx.servState.channelList == nil {
		ctx.servState.channelList = NewChannelList(ctx.servConn, ctx.servState)
	}
	if ctx.servState.channelList.complete {
		ctx.servState.channelList.Clear()
	}
	if !ctx.servState.channelList.inProgress {
		ctx.servConn.conn.Raw("LIST")
	}
}

const SCRIPTS_DIR = "./scripts/"

var scriptAliases = map[string]string{}

func registerCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		ctx.tab.Println("usage: /register [alias] [script file]")
		return
	}

	name := args[0]
	file := args[1]

	if _, ok := clientCommands[name]; ok {
		if alias, ok := scriptAliases[name]; ok {
			ctx.tab.Println("overwriting previous alias of " + alias + " for /" + name)
		} else {
			ctx.tab.Println("ERROR: cannot overwrite built-in client command /" + name)
			return
		}
	}

	scriptAliases[name] = file
	clientCommands[name] = func(ctx *commandContext, args ...string) {
		scriptCmd(ctx, append([]string{file}, args...)...)
	}
	ctx.tab.Println("/" + name + " registered as alias to " + file)
}

func unregisterCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		ctx.tab.Println("usage: /unregister [alias]")
		return
	}
	name := args[0]
	alias, ok := scriptAliases[name]
	if !ok {
		ctx.tab.Println("ERROR: /" + name + " is not a registered alias of any script")
		return
	}
	delete(scriptAliases, name)
	delete(clientCommands, name)
	ctx.tab.Println("/" + name + " (" + alias + ") unregistered")
}

func scriptCmd(ctx *commandContext, args ...string) {
	if len(args) < 0 {
		ctx.tab.Println("usage: /script [file in " + SCRIPTS_DIR + "] [args...]")
		return
	}

	if ctx.chanState == nil && ctx.pmState == nil {
		ctx.tab.Println("ERROR: scripts only work in channels and private messages!")
		return
	}

	scriptFile := filepath.Base(args[0])

	f, err := os.Open(SCRIPTS_DIR + scriptFile)
	checkErr(err)
	if os.IsNotExist(err) {
		ctx.tab.Println("script not found: " + scriptFile)
		return
	}
	{
		finfo, err := f.Stat()
		checkErr(err)
		if finfo.IsDir() {
			ctx.tab.Println("script not found: " + scriptFile)
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
		ctx.tab.Println("unsupported script type: " + ext)
		return
	}

	go func() {
		cmd := exec.Command(bin, args...)
		stdout := bytes.NewBuffer([]byte{})
		stderr := bytes.NewBuffer([]byte{})
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			ctx.tab.Println("ERROR: " + bin + " " + strings.Join(args, " ") + "\r\nreturned: " + err.Error())
		}
		out := strings.TrimSpace(stdout.String())
		err := strings.TrimSpace(stderr.String())
		if err != "" {
			ctx.tab.Println(bin + " " + strings.Join(args, " ") + "\r\nreturned: " + err)
		}
		if out != "" {
			ctx.tab.Send(out)
		}
	}()
}
