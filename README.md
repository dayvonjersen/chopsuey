![[](#contributing-and-feedback)](https://img.shields.io/badge/feedback-welcome-%23ff086e.svg)

# ![](chopsuey.ico) chopsuey

    an IRC client for Windows

(screenshots go here)

contents:
                       [Downloads](#download) |
                        [Features](#features) |
                [Known Issues](#known-issues) |
               [/commands](#list-of-commands) |
    [Keyboard Shortcuts](#keyboard-shortcuts) |
[Building from Source](#building-from-source) |
[Contributing](#contributing-and-feedback)

### Download

 - ## [LATEST VERSION: 0.7[BETA] Windows 64-bit (x86_64) :point_left: ](about:blank)
   - (*no win32 build sorry*)

##### Previous Releases:

 - 0.7[PRE-ALPHA]: we don't talk about it

### Getting Started

 Download the zip file and extract the contents. Open the `chopsuey` folder you
 just created and double-click on the `.exe` file with an icon of a chinese takeout box.
 Then in the window that just opened use your keyboard and:

 - type `/server irc.rizon.net +6697` and press `Enter`

 - type `/join #/g/punk` and press `Enter`

 - type `tso: this is great I love this client so much thank you <3` and press `Enter`

### Features

 - lightweight, fast, free

 - everything you need, nothing you don't

 - customizable <sup>[1](#customization)</sup>

 - scripting maybe <sup>[2](#scripting)</sup>

### Known Issues

 - shit's broke yo

### List of Commands

 #### /connect
reconnect to server (if disconnected) (specify with **/server**)


#### /disconnect

disconnect from server and do not try to reconnect


#### /quit

disconnect from server and close all associated tabs (sends quit message)


#### /reconnect

disconnect and reconnect to server (specify with **/server**)


#### /server [host]  [+][port (default 6667, ssl 6697)]

open a connection to an irc network e.g. `/server irc.example.org`

to use ssl prefix port number with + e.g. `/server irc.example.org +6697`

#### /clear

remove all text from the current buffer

#### /close [part or quit message]

closes current tab with optional part or quit message

if on a channel, same as **/part**

if on a server same as **/quit**


#### /ctcp [nick] [message] [args...]

send a CTCP message to nick with optional arguments

#### /join [#channel]

attempt to join a channel, opens a new tab if successful

#### /kick [nick] [(optional) reason...]

remove a user from a channel (if you have op)

#### /list

opens a tab with all the channels on the server in a sortable table view

double click on a channel to try to join it

#### /me [message...]

\*tso slaps you around with a big trout\*

#### /mode [#channel or your nick] [mode] [nicks...]

set one or more modes for a channel or one or more nicks

#### /msg [nick] [message...]

opens a new tab and send a private message

#### /nick [new nick]

change your handle

#### /notice [#channel or nick] [message...]

sends a NOTICE. *please dont send NOTICEs to channels...*

#### /part [message...]

leave a channel with optional message (**and currently closes the tab but I'm going to change this in the future**)

#### /rejoin

join a channel you have left (because of having been disconnected, kicked or having parted

#### /topic [new topic...]

set or view the topic for the channel if you have permission to do so

#### /version [nick]

find out what client someone is using

#### /whois [nick]

send a whois query to server

#### /away [message]
mark yourself as being (Away)!

#### /unaway
announce your triumphant return

#### /help [command]

`/help` produces a list of all available commands with a brief summary of their usage

`/help [command]` shows usage about a specific command

#### /exit

exits the application

#### /script [file in `scripts/`] [args...]

run an external program and send its output as a message in the current channel or private message tab

recognized filetypes: (**iff you have the associated interpreter installed on your system**):

| interpreter  | extension |
|--------------|-----------|
| go run       | `.go`     |
| php          | `.php`    |
| perl         | `.pl`     |
| python       | `.py`     |
| ruby         | `.rb`     |
| bash         | `.sh`     |

#### /call [file in `scripts/`] [args...]

alias of `/script`

#### /register [alias] [script file]

alias a script to a command you can call directly e.g.
```
/register mycommand cool_script.pl
/mycommand hey guys
```
is synonymous with:
`/script cool_script.pl hey guys`

#### /unregister [alias]

unalias a command registered with `/register`

#### /theme [file in `themes/`]

change colors

#### /font [font name that might have spaces] [font size]

change font (*destroys previous text colors in buffer currently*)

#### /palette

(screenshot here)

### Keyboard Shortcuts

#### application

 - `ctrl+q` exit application
 - `shift+tab` toggle window border
 - `f2` make window more transparent (enables transparency)
 - `f3` enable/disable transparency
 - `f4` make window less transparent (enables transparency)

#### tab navigation

 - `ctrl+t` opens a new empty server tab. use `/server` to connect
 - `ctrl+tab` changes focus to the tab on the right
 - `ctrl+shift+tab` changes focus to the tab on the left
 - `ctrl+f4` and `ctrl+w` close the current tab (if a server, it will close all associated tabs as well)

#### text formatting
 - `ctrl+k` inserts color code control character. specify colors with numbers e.g.

 `[ctrl+k]4this text is red`

 `[ctrl+k]0,4this text is white with a red background`

| color     | number |
|-----------|--------|
| White     | 0      |
| Black     | 1      |
| Navy      | 2      |
| Green     | 3      |
| Red       | 4      |
| Maroon    | 5      |
| Purple    | 6      |
| Orange    | 7      |
| Yellow    | 8      |
| Lime      | 9      |
| Teal      | 10     |
| Cyan      | 11     |
| Blue      | 12     |
| Pink      | 13     |
| DarkGray  | 14     |
| LightGray | 15     |

 - `ctrl+b` insert bold control code character. insert again to toggle.

 `[ctrl+b]this will be bold[ctrl+b] this won't` == **this will be bold** this won't

 - `ctrl+i` insert italic control code character. insert again to toggle.

 `[ctrl+i]this will be italic[ctrl+i] this won't` == *this will be italic* this won't

 - `ctrl+u` insert underline control code character. insert again to toggle.

 `[ctrl+u]this will be underlined[ctrl+u] this won't` == (github markdown doesn't have underline)

 - `ctrl+s` insert strikethrough control code character. insert again to toggle.

  `[ctrl+s]this will be strikethrough[ctrl+b] this won't` == ~~this will be strikethrough~~ this won't

 - `ctrl+0`: reset formatting to default

### Building from Source

>NOTE(tso): This is for **64-bit windows-only**.
It might compile on other systems if you set `GOOS` and `GOARCH`
but it won't run. I have not tested it in wine or on ReactOS or any version of
Windows other than 7SP1.

```
mkdir -p $GOPATH/src/github.com
cd !:2
git clone git@github.com:generaltso/chopsuey
cd chopsuey
go get github.com/lxn/walk
go get github.com/fluffle/goirc
go get github.com/akavel/rsrc
go get github.com/maruel/panicparse/cmd/pp
go get github.com/kr/pretty
make icon
make
```

If that doesn't work [tell me about it](mailto:tso@teknik.io?Subject=it%20doesnt%20work).

### Contributing and Feedback

If you want to help me make this better [here's my TODO list](https://github.com/generaltso/chopsuey/blob/master/TODO.txt).

You can also `git grep` for these tags: `NOTE` `TODO` `FIXME` `HACK` `WTF`

Feel free to [open an issue]() or [submit a pull request]() to start a discussion.

I welcome any and all constructive criticism and feedback. [E-mail me](mailto:tso@teknik.io?Subject=sup) or leave me a message on IRC (I'm tso on [Rizon](irc://irc.rizon.net:6697/) and tzo [freenode](irc://irc.freenode.net:6697/))
I'll try to respond if I'm awake and my client doesn't crash.
