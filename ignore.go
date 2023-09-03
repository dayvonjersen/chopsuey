/*
NOTE(tso): this is an awful hackjob
  - couldn't pick a good name for a "list of nicks"
    because nickList is already a thing
  - but nickList *SHOULD* actually be adapted for use instead of this shit
    with the application of just a little critical thinking
    which I'm apparently incapable of atm
  - function signatures are inconsistent af because see above
  - ignoreList should actually be per-server or even per-channel
  - ignoreList should accept patterns *!@*.* whatever the syntax is i forget
  - the type of ignoreList should be reusable for a "banList", "inviteList", ...
  - even though we're using pointer values here we have to update name/host
    manually because we replace the concrete value in the nickList for some
    unknown reason I have to investigate why probably something to do with
    prefixes which are fucked anyway
  - scratch that we're using values because everything is terrible
  - rushing to implement this in a suboptimal way because
    I'm sick of people brewing soykaf and being forced to look at it
    -tso 2018-08-29 04:03:16a
*/
package main

import "sync"

var ignoreList = newListerine()

type listerine struct {
	list []nick
	mu   *sync.Mutex
}

func newListerine() *listerine {
	return &listerine{
		list: []nick{},
		mu:   &sync.Mutex{},
	}
}

func (l *listerine) Add(n nick) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.list = append(l.list, n)
}

func (l *listerine) Remove(nick string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, n := range l.list {
		if n.name == nick {
			l.list = append(l.list[0:i], l.list[i+1:]...)
			return
		}
	}
}

func (l *listerine) UpdateHost(name, host string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, n := range l.list {
		if n.name == name {
			l.list[i].host = host
		}
	}
}

func (l *listerine) UpdateNick(oldNick, newNick string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, n := range l.list {
		if n.name == oldNick {
			l.list[i].name = newNick
		}
	}
}

func (l *listerine) Has(name, host string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range l.list {
		if n.name == name || n.host == host {
			return true
		}
	}
	return false
}
