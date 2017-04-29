package main

import (
	"log"
	"regexp"
	"sort"
	"sync"
)

type nick struct {
	prefix, name string
}

func (n *nick) String() string {
	return n.prefix + n.name
}

var nickRegex = regexp.MustCompile("^([~&@%+]*)(.+)$")

func newNick(prefixed string) *nick {
	m := nickRegex.FindAllStringSubmatch(prefixed, -1)
	return &nick{m[0][1], m[0][2]}
}

type nickList struct {
	data   []string         // underlying sorted string array for searching
	lookup map[string]*nick // lookup map for prefixes
	mu     *sync.Mutex
}

func newNickList() *nickList {
	nl := &nickList{
		data:   []string{},
		lookup: map[string]*nick{},
		mu:     &sync.Mutex{},
	}
	return nl
}

func (nl *nickList) Add(n string) {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	nick := newNick(n)

	i := sort.SearchStrings(nl.data, nick.name)
	if i < len(nl.data) {
		if nl.data[i] != nick.name {
			nl.data = append(nl.data[:i], append([]string{nick.name}, nl.data[i:]...)...)
			nl.lookup[nick.name] = nick
		} else if nick.prefix != "" && nl.lookup[nick.name].prefix != nick.prefix {
			nl.lookup[nick.name] = nick
		}
	} else {
		nl.data = append(nl.data, nick.name)
		nl.lookup[nick.name] = nick
	}
}

func (nl *nickList) Remove(n string) {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	if nl.Has(n) {
		nick := newNick(n)
		i := sort.SearchStrings(nl.data, nick.name)
		nl.data = append(nl.data[0:i], nl.data[i+1:]...)
		delete(nl.lookup, n)
	}
}

func (nl *nickList) Has(n string) bool {
	nick := newNick(n)
	i := sort.SearchStrings(nl.data, nick.name)
	return i < len(nl.data) && nl.data[i] == nick.name
}

func (nl *nickList) Get(n string) *nick {
	nick, ok := nl.lookup[n]
	if !ok {
		panic("nick \"" + n + "\" not in lookup table")
	}
	return nick
}

func (nl *nickList) Set(n string, newNick *nick) {
	nl.mu.Lock()
	oldNick, ok := nl.lookup[n]
	nl.mu.Unlock()
	if !ok {
		panic("nick \"" + n + "\" not in lookup table")
	}
	if oldNick.name != newNick.name {
		nl.Remove(oldNick.name)
		nl.Add(newNick.String())
	} else if oldNick.prefix != newNick.prefix {
		nl.mu.Lock()
		nl.lookup[n] = newNick
		nl.mu.Unlock()
	}
}

func (nl *nickList) StringSlice() []string {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	n := nickListByPrefix{}
	for _, nick := range nl.lookup {
		nick.prefix = sortPrefix(nick.prefix)
		n = append(n, nick)
	}
	sort.Sort(n)
	s := []string{}
	for _, nick := range n {
		s = append(s, nick.String())
	}
	return s
}

func cmpPrefix(a, b byte) bool {
	switch a {
	case '~':
		return true
	case '&':
		return b != '~'
	case '@':
		return b != '~' && b != '&'
	case '%':
		return b != '~' && b != '&' && b != '@'
	case '+':
		return false
	}
	panic("unhandled prefix: " + string(a))
	return false
}

func sortPrefix(prefix string) string {
	s := []byte(prefix)
	sort.Slice(s, func(i, j int) bool {
		a, b := s[i], s[j]
		return cmpPrefix(a, b)
	})
	return string(s)
}

type nickListByPrefix []*nick

func (nl nickListByPrefix) Len() int {
	return len(nl)
}

func (nl nickListByPrefix) Less(i, j int) bool {
	a, b := nl[i].prefix, nl[j].prefix
	if a == b {
		return nl[i].name < nl[j].name
	}
	if len(a) == 0 {
		return false
	}
	if len(b) == 0 {
		return true
	}

	return cmpPrefix(a[0], b[0])
}

func (nl nickListByPrefix) Swap(i, j int) {
	nl[i], nl[j] = nl[j], nl[i]
}
