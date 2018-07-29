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

### Keyboard Shortcuts

### Building from Source

NOTE(tso): This is for **64-bit windows-only**.
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
