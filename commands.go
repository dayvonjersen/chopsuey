package main

import (
	"bytes"
	"fmt"
	"image"
	col "image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type commandContext struct {
	servConn *serverConnection
	tab      tabWithTextBuffer

	servState *serverState
	chanState *channelState
	pmState   *privmsgState
}

type (
	clientCommand    func(ctx *commandContext, args ...string)
	clientCommandDoc struct {
		usage, desc string
	}
)

var (
	clientCommands   map[string]clientCommand
	clientCommandDox = map[string]clientCommandDoc{
		// connectivity
		"connect":    clientCommandDoc{"/connect", "(if disconnected) reconnect to server\nspecify server with /server"},
		"disconnect": clientCommandDoc{"/disconnect", "disconnect from server and do not try to reconnect"},
		"quit":       clientCommandDoc{"/quit", "disconnect from server and close all associated tabs"},
		"reconnect":  clientCommandDoc{"/recover", "disconnect and reconnect to server\nspecify server with /server"},
		"server": clientCommandDoc{"/server [host]  [+][port (default 6667, ssl 6697)]",
			"open a connection to an irc network, to use ssl prefix port number with +"},

		// other core functionality
		"clear": clientCommandDoc{"/clear", "remove all text from the current buffer"},
		"close": clientCommandDoc{"/close [part or quit message]",
			"closes current tab with optional part or quit message\nif on a channel, same as /part\nif on a server same as /quit"},
		"ctcp": clientCommandDoc{"/ctcp [nick] [message] [args...]", "send a CTCP message to nick with optional arguments"},
		"join": clientCommandDoc{"/join [#channel]", "join a channel"},
		"kick": clientCommandDoc{"/kick [nick] [(optional) reason...]", "remove a user from a channel (if you have op)"},
		"list": clientCommandDoc{"/list", "opens a tab with all the channels on the server"},
		"me":   clientCommandDoc{"/me [message...]", "*tso slaps you around with a big trout*"},
		"mode": clientCommandDoc{"/mode [#channel or your nick] [mode] [nicks...]",
			"set one or more modes for a channel or one or more nicks"},
		"msg":  clientCommandDoc{"/msg [nick] [message...]", "opens a new tab and send a private message"},
		"nick": clientCommandDoc{"/nick [new nick]", "change your handle"},
		"notice": clientCommandDoc{"/notice [#channel or nick] [message...]",
			"sends a NOTICE. please dont send NOTICEs to channels..."},
		"part":   clientCommandDoc{"/part [message...]", "(doesnt close tab) leave a channel with optional message"},
		"rejoin": clientCommandDoc{"/rejoin", "join a channel you have left (either by being kicked or having parted)"},
		"topic":  clientCommandDoc{"/topic [new topic...]", "set or view the topic for the channel"},

		"version": clientCommandDoc{"/version [nick]", "find out what client someone is using"},
		"whois":   clientCommandDoc{"/whois [nick]", "find out a user's true identity"},

		// stuff no one uses anymore
		"away":   clientCommandDoc{"/away [message]", "mark yourself as being (Away)!"},
		"unaway": clientCommandDoc{"/unaway", "announce your triumphant return"},

		// stubs
		"help": clientCommandDoc{"/help [command]", "..."},
		"exit": clientCommandDoc{"/exit", "SHUT\nIT\nDOWN"},

		// experimental/WIP
		"send": clientCommandDoc{"/send [nick] [filepath (optional)]",
			"offer to send a file to a user, if no file is specified a dialog will open to pick one\n" +
				"please note that file transfers dont work in all clients\n" +
				"and file sharing may be prohibited by the network you are on"},

		// scripting
		"script": clientCommandDoc{"/script [file in " + SCRIPTS_DIR + "] [args...]",
			"run an external program and send its output as a message in the current context\n" +
				"recognized filetypes (" + bold("iff you have the associated interpreter installed") + "):\n" +
				"go, php, perl, python, ruby, and bash"},
		"register": clientCommandDoc{"/register [alias] [script file]",
			"alias a script to a command you can call directly e.g.\n" +
				"/register mycommand cool_script.pl\nmakes\n" +
				"/mycommand hey guys\nsynonymous with\n/script cool_script.pl hey guys"},
		"unregister": clientCommandDoc{"/unregister [alias]", "unalias a command registered with /register"},
	}
	scriptAliases = map[string]string{}
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
		"clear":   clearCmd,
		"close":   closeCmd,
		"ctcp":    ctcpCmd,
		"join":    joinCmd,
		"kick":    kickCmd,
		"list":    listCmd,
		"me":      meCmd,
		"mode":    modeCmd,
		"msg":     privmsgCmd,
		"privmsg": privmsgCmd,
		"nick":    nickCmd,
		"notice":  noticeCmd,
		"part":    partCmd,
		"rejoin":  rejoinCmd,
		"topic":   topicCmd,
		"version": versionCmd,
		"whois":   whoisCmd,

		// stuff no one uses anymore
		"away":   awayCmd,
		"unaway": unawayCmd,

		// stubs
		"help": helpCmd,
		"exit": exitCmd,

		// experimental/WIP
		"send": sendCmd,

		// scripting
		"call":       scriptCmd,
		"script":     scriptCmd,
		"register":   registerCmd,
		"unregister": unregisterCmd,

		// developer commands do not use
		"context":    contextCmd,
		"font":       fontCmd,
		"palette":    paletteCmd,
		"raw":        rawCmd,
		"screenshot": screenshotCmd,
		"theme":      themeCmd,

		// harmful
		"ignore":   ignoreCmd,
		"unignore": unignoreCmd,
	}
}

// helpers
func usage(ctx *commandContext, cmd string) {
	clientMessage(ctx.tab, clientCommandDox[cmd].usage)
}

func requireServConn(ctx *commandContext) bool {
	if ctx.servState.connState != CONNECTED {
		clientError(ctx.tab, "not connected to any network!!!")
		return false
	}
	return true
}

// commands
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
	if !requireServConn(ctx) {
		return
	}
	disconnectCmd(ctx, args...)

	tabMan.Delete(tabMan.FindAll(allServerTabsFinder(ctx.servState))...)

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
	ctx.servState.tab.Close()

	if tabMan.Len() == 0 {
		empty := newEmptyServerTab()
		empty.servState = ctx.servState
		empty.servConn = nil
	}

	SetSystrayContextMenu()
}

func reconnectCmd(ctx *commandContext, args ...string) {
	switch ctx.servState.connState {
	case CONNECTION_EMPTY:
		clientMessage(ctx.tab, "ERROR: no network specified (use /server)")
	case CONNECTING, CONNECTION_START:
		clientMessage(ctx.tab, "ERROR: connection in progress: "+serverAddr(ctx.servState.hostname, ctx.servState.port))

	case CONNECTED:
		disconnectCmd(ctx, args...)
		fallthrough
	case DISCONNECTED, CONNECTION_ERROR:
		ctx.servConn.retryConnectEnabled = true
		ctx.servConn.cancelRetryConnect = make(chan struct{})
		connectCmd(ctx, args...)
	}
}

func serverCmd(cmdctx *commandContext, args ...string) {
	if len(args) < 1 {
		usage(cmdctx, "server")
		return
	}

	// TODO(tso): abstract opening a new server connection/tab by reusing this code
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
	servState.connState = CONNECTION_EMPTY
	servState.hostname = hostname
	servState.port = port
	servState.ssl = ssl
	servState.networkName = serverAddr(hostname, port)
	servState.user = &userState{
		nick: cmdctx.servState.user.nick,
	}
	servState.channels = map[string]*channelState{}
	servState.privmsgs = map[string]*privmsgState{}

	servConn := NewServerConnection(servState, func() {})

	ctx := tabMan.CreateIfNotFound(&tabContext{servConn: servConn, servState: servState}, tabMan.Len(), func(t *tabWithContext) bool {
		return t.servConn == nil
	})
	ctx.servConn = servConn
	ctx.servState = servState

	if ctx.tab == nil {
		tab := newServerTab(servConn, servState)
		ctx.tab = tab
	}
	servState.tab = ctx.tab.(*tabServer)

	servConn.Connect(servState)
}

func clearCmd(ctx *commandContext, args ...string) {
	ctx.tab.Clear()
}

func closeCmd(ctx *commandContext, args ...string) {
	if ctx.tab == ctx.servState.tab {
		if ctx.servState.connState == CONNECTED {
			quitCmd(ctx, args...)
		}

		if tabMan.Len() == 1 {
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
	tabCtx := &tabWithContext{tab: ctx.tab}
	tabCtx.servConn = ctx.servConn
	tabCtx.servState = ctx.servState
	tabCtx.chanState = ctx.chanState
	tabCtx.pmState = ctx.pmState
	tabMan.Delete(tabCtx)
	ctx.tab.Close()
	SetSystrayContextMenu()
}

func ctcpCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 2 {
		usage(ctx, "ctcp")
		return
	}
	ctx.servConn.conn.Ctcp(args[0], args[1], args[2:]...)
}

func joinCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) != 1 || len(args[0]) < 2 || args[0][0] != '#' {
		usage(ctx, "join")
		return
	}
	ctx.servConn.conn.Join(args[0])
}

func kickCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 1 {
		usage(ctx, "kick")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Kick(ctx.chanState.channel, args[0], args[1:]...)
	} else {
		clientError(ctx.tab, "/kick only works for channels.")
	}
}

func listCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
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
	if !requireServConn(ctx) {
		return
	}
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		usage(ctx, "me")
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
	if !requireServConn(ctx) {
		return
	}
	// NOTE(tso): /mode
	// /mode                -> channel is assumed to be chanState.channel
	// /mode nick           -> channel is assumed to be chanState.channel
	// /mode #channel       -> chanState can be nil and not be current channel
	// /mode ±abcd          -> channel is assumed to be chanState.channel
	// /mode ±abcd #channel -> chanState can be nil and not be current channel
	// /mode ±abcd #channel1 #channel2 ... -> invalid
	// /mode ±abcd nick1 nick2...          -> valid
	// @_@
	// -tso 7/14/2018 8:42:22 AM

	if len(args) < 1 {
		usage(ctx, "mode")
		return
	}
	ctx.servConn.conn.Mode(args[0], args[1:]...)
}

func privmsgCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 2 || isChannel(args[0]) {
		usage(ctx, "msg")
		return
	}
	nick := args[0]
	msg := strings.Join(args[1:], " ")

	if isService(nick) {
		ctx.servConn.conn.Privmsg(nick, msg)
		noticeMessage(
			ctx.servState.CurrentTab(),
			ctx.servState.user.nick, nick, msg)
		return
	}

	pmState := ensurePmState(ctx.servConn, ctx.servState, nick)

	ctx.servConn.conn.Privmsg(nick, msg)
	mw.WindowBase.Synchronize(func() {
		privateMessage(pmState.tab, ctx.servState.user.nick, msg)
		checkErr(tabWidget.SetCurrentIndex(pmState.tab.Index()))
	})
}

func nickCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) != 1 {
		usage(ctx, "nick")
		return
	}
	ctx.servConn.conn.Nick(args[0])
}

func noticeCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 2 {
		usage(ctx, "notice")
		return
	}
	msg := strings.Join(args[1:], " ")
	ctx.servConn.conn.Notice(args[0], msg)
	noticeMessage(ctx.tab, ctx.servState.user.nick, args[0], msg)
}

func partCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, strings.Join(args, " "), ctx.servState)
	} else {
		clientError(ctx.tab, "ERROR: /part only works for channels. Try /close")
	}
}

func rejoinCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.Part(ctx.chanState.channel, "rejoining...", ctx.servState)
		ctx.servConn.conn.Join(ctx.chanState.channel)
	} else {
		clientError(ctx.tab, "ERROR: /rejoin only works for channels.")
	}
}

func topicCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 1 {
		usage(ctx, "topic")
		return
	}
	if ctx.chanState != nil {
		ctx.servConn.conn.Topic(ctx.chanState.channel, args...)
	} else {
		clientError(ctx.tab, "ERROR: /topic can only be used in channels")
	}
}

func versionCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) != 1 {
		usage(ctx, "version")
		return
	}
	ctx.servConn.conn.Version(args[0])
}

func whoisCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) != 1 {
		usage(ctx, "whois")
		return
	}
	ctx.servConn.conn.Whois(args[0])
}

func awayCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	msg := strings.Join(args, " ")
	if len(args) == 0 {
		msg = "(Away!)"
	}
	ctx.servConn.conn.Away(msg)
}

func unawayCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	ctx.servConn.conn.Away()
}

func helpCmd(ctx *commandContext, args ...string) {
	if len(args) > 0 {
		for n, cmd := range args {
			if n > 0 {
				clientMessage(ctx.tab, "\n")
			}
			cmd = strings.Trim(cmd, "/")
			if dox, ok := clientCommandDox[cmd]; ok {
				clientMessage(ctx.tab, dox.desc)
				clientMessage(ctx.tab, "usage:", dox.usage)
			} else {
				clientError(ctx.tab, "no such command:", cmd)
				clientMessage(ctx.tab, "use /help with no arguments for list of available commands")
			}
		}
	} else {
		clientMessage(ctx.tab, "\n", bold(color("COMMANDS:", Blue)))
		clientMessage(ctx.tab, color("type /help [command] for a short description.", LightGrey))
		yaymaps := []string{}
		padlen := 0
		for cmd := range clientCommandDox {
			yaymaps = append(yaymaps, cmd)
			if len(cmd) > padlen {
				padlen = len(cmd)
			}
		}
		sort.Strings(yaymaps)
		for _, cmd := range yaymaps {
			usage := strings.SplitN(clientCommandDox[cmd].usage, " ", 2)
			clientMessage(ctx.tab, append([]string{usage[0] + strings.Repeat(" ", padlen-len(cmd))}, usage[1:]...)...)
		}
		clientMessage(ctx.tab, "\n")
		if len(scriptAliases) > 0 {
			clientMessage(ctx.tab,
				color("SCRIPT ALIASES:", Blue)+"\n"+
					color("to remove an alias type /unregister [alias]", LightGrey))

			yaymaps := []string{}
			for alias := range scriptAliases {
				yaymaps = append(yaymaps, alias)
			}
			sort.Strings(yaymaps)
			for _, alias := range yaymaps {
				clientMessage(ctx.tab, "/"+alias, "=>", scriptAliases[alias])
			}
			clientMessage(ctx.tab, "\n")
		}
	}
}

func exitCmd(ctx *commandContext, args ...string) {
	exit()
}

func sendCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	who := args[0]
	file := "rfc2812.txt"
	fileTransfer(ctx.servConn, who, file)
}

func scriptCmd(ctx *commandContext, args ...string) {
	if !requireServConn(ctx) {
		return
	}
	if len(args) < 0 {
		usage(ctx, "script")
		return
	}

	if ctx.chanState == nil && ctx.pmState == nil {
		clientMessage(ctx.tab, "ERROR: scripts only work in channels and private messages!")
		return
	}

	scriptFile := filepath.Base(args[0])

	f, err := os.Open(SCRIPTS_DIR + scriptFile)
	if os.IsNotExist(err) {
		clientError(ctx.tab, "script not found: "+scriptFile)
		return
	}
	checkErr(err)
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
			switch ctx.tab.(type) {
			case *tabChannel:
				ctx.tab.(*tabChannel).Send(out)
			case *tabPrivmsg:
				ctx.tab.(*tabPrivmsg).Send(out)
			default:
				clientMessage(ctx.tab, color("OUTPUT("+scriptFile+")", White, Green)+":", out)
			}
		}
	}()
}

func registerCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		usage(ctx, "register")
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
		usage(ctx, "register")
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

func contextCmd(ctx *commandContext, args ...string) {
	servConn := serverConnection{}
	servConnHasConn := false
	if ctx.servConn != nil {
		servConn = *ctx.servConn
		servConnHasConn = ctx.servConn.conn != nil
	}
	servState := serverState{}
	servStateHasTab := false
	if ctx.servState != nil {
		servState = *ctx.servState
		servStateHasTab = ctx.servState.tab != nil
		servState.tab = nil
	}

	log.Println("pointers:\n\t", strings.Join(strings.Split(fmt.Sprintf("%#v", ctx), ", "), "\n\t"))
	log.Println("servConn:")
	printf(servConn)
	log.Println("servConn has goirc.Conn:", servConnHasConn)
	log.Println("servState:")
	printf(servState)
	log.Println("servState has serverTab:", servStateHasTab)
}

func fontCmd(ctx *commandContext, args ...string) {
	if len(args) < 2 {
		return
	}
	face := strings.Join(args[:len(args)-1], " ")
	size, _ := strconv.Atoi(args[len(args)-1])

	font, err := walk.NewFont(face, size, 0)
	checkErr(err)

	mw.WindowBase.SetFont(font)
}

func paletteCmd(ctx *commandContext, args ...string) {
	clientMessage(ctx.tab, "palette test:")
	for c := 0; c < 16; c++ {
		str := fmt.Sprintf("%02d:% 9s", c, colorCodeString(c))
		Println(CUSTOM_MESSAGE, T(ctx.tab),
			color(str, White, c),
			color(str, Black, c),
			color(str, c),
		)
	}
}

func rawCmd(ctx *commandContext, args ...string) {
	// if !requireServConn(ctx) {
	// 	return
	// }
	ctx.servConn.conn.Raw(strings.Join(args, " "))
}

func screenshotCmd(ctx *commandContext, args ...string) {
	go func() {
		// t. stackoverflow
		// https://stackoverflow.com/a/3291411

		// get the device context of the screen
		// hScreenDC := win.CreateDC(syscall.StringToUTF16Ptr("DISPLAY"), nil, nil, nil)
		hScreenDC := win.GetDC(win.GetParent(mw.Handle()))
		// and a device context to put it in
		hMemoryDC := win.CreateCompatibleDC(hScreenDC)

		// width := win.GetDeviceCaps(hScreenDC, win.HORZRES)
		// height := win.GetDeviceCaps(hScreenDC, win.VERTRES)
		rect := mw.Bounds()
		width := int32(rect.Width)
		height := int32(rect.Height)

		// maybe worth checking these are positive values
		hBitmap := win.CreateCompatibleBitmap(hScreenDC, width, height)

		// get a new bitmap
		hOldBitmap := win.HBITMAP(win.SelectObject(hMemoryDC, win.HGDIOBJ(hBitmap)))

		win.BitBlt(hMemoryDC, 0, 0, width, height, hScreenDC, int32(rect.X), int32(rect.Y), win.SRCCOPY)
		hBitmap = win.HBITMAP(win.SelectObject(hMemoryDC, win.HGDIOBJ(hOldBitmap)))

		// clean up
		win.DeleteDC(hMemoryDC)
		win.DeleteDC(hScreenDC)

		{
			// t. lxn
			// http://localhost/src/github.com/lxn/walk/bitmap.go?s=1439:1495#L148

			var bi win.BITMAPINFO
			bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
			hdc := win.GetDC(0)
			if ret := win.GetDIBits(hdc, hBitmap, 0, 0, nil, &bi, win.DIB_RGB_COLORS); ret == 0 {
				panic("GetDIBits get bitmapinfo failed")
			}

			buf := make([]byte, bi.BmiHeader.BiSizeImage)
			bi.BmiHeader.BiCompression = win.BI_RGB
			if ret := win.GetDIBits(hdc, hBitmap, 0, uint32(bi.BmiHeader.BiHeight), &buf[0], &bi, win.DIB_RGB_COLORS); ret == 0 {
				panic("GetDIBits failed")
			}

			width := int(bi.BmiHeader.BiWidth)
			height := int(bi.BmiHeader.BiHeight)
			img := image.NewRGBA(image.Rect(0, 0, width, height))

			n := 0
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					r := buf[n+2]
					g := buf[n+1]
					b := buf[n+0]
					n += int(bi.BmiHeader.BiBitCount) / 8
					img.Set(x, height-y-1, col.RGBA{r, g, b, 255})
				}
			}

			fname := SCREENSHOTS_DIR + time.Now().Format("20060102150405.999999999") + ".png"
			f, err := os.Create(fname)
			checkErr(err)
			png.Encode(f, img)
			f.Close()

			abspath, err := filepath.Abs(fname)
			checkErr(err)
			clientMessage(ctx.tab, now(), "screenshot:", "file:///"+strings.Replace(abspath, "\\", "/", -1))

			if !requireServConn(ctx) {
				return
			}
			if ctx.chanState != nil || ctx.pmState != nil {
				// FIXME(tso): stop being lazy and do this with net/http instead
				cmd := exec.Command("curl", "-silent", "-F", "file=@"+fname, "https://0x0.st")
				out, err := cmd.CombinedOutput()
				if err == nil {
					lines := strings.Split(strings.TrimSpace(string(out)), "\n")
					url := lines[len(lines)-1]
					if ctx.chanState != nil {
						ctx.tab.(*tabChannel).Send(url)
					}
					if ctx.pmState != nil {
						ctx.tab.(*tabPrivmsg).Send(url)
					}
				} else {
					clientError(ctx.tab, string(out))
				}
			}
		}
	}()
}

func themeCmd(ctx *commandContext, args ...string) {
	if err := applyTheme(args[0]); err != nil {
		clientError(ctx.tab, "error applying theme:", err.Error())
	}
}

func ignoreCmd(ctx *commandContext, args ...string) {
	if ctx.chanState == nil || len(args) != 1 {
		return
	}

	nick := ctx.chanState.nickList.Get(args[0])
	if nick == nil {
		return
	}

	ignoreList.Add(*nick)
}

func unignoreCmd(ctx *commandContext, args ...string) {
	if len(args) != 1 {
		return
	}

	ignoreList.Remove(args[0])
}
