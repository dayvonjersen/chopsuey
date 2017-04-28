package main

import (
	"fmt"
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

func splitNick(prefixed string) *nick {
	m := nickRegex.FindAllStringSubmatch(prefixed, -1)
	return &nick{m[0][1], m[0][2]}
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

type nickList struct {
	slice []*nick
	Mu    *sync.Mutex
}

func (nl *nickList) Len() int {
	return len(nl.slice)
}
func (nl *nickList) Less(i, j int) bool {
	return nl.slice[i].name < nl.slice[j].name
}
func (nl *nickList) Swap(i, j int) {
	nl.slice[i], nl.slice[j] = nl.slice[j], nl.slice[i]
}

func (nl *nickList) FindIndex(n *nick) int {
	return nl.FindIndexSelection(n)

	// return sort.Search(nl.Len(), func(i int) bool { return nl.slice[i].name == n.name })
}

func (nl *nickList) FindIndexSelection(n *nick) int {
	for i, o := range nl.slice {
		if o.name == n.name {
			return i
		}
	}
	return nl.Len()
}

func (nl *nickList) FindIndexBinary(n *nick) int {
	i, j := 0, nl.Len()-1
	for i <= j {
		k := (i + j) / 2
		o := nl.slice[k].name
		if o > n.name {
			j = k - 1
		} else if o < n.name {
			i = k + 1
		} else {
			return k
		}
	}
	return nl.Len()
}

func (nl *nickList) Has(prefixed string) bool {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < nl.Len() && nl.slice[i].name == n.name {
		return true
	}
	return false
}

func (nl *nickList) Add(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	n.prefix = sortPrefix(n.prefix)
	if i < nl.Len() && nl.slice[i].name == n.name {
		if nl.slice[i].prefix != n.prefix {
			nl.slice[i].prefix = n.prefix
		}
	} else if i < nl.Len() {
		nl.slice = append(nl.slice[:i], append([]*nick{n}, nl.slice[i:]...)...)
	} else {
		nl.slice = append(nl.slice, n)
	}
	if !sort.IsSorted(nl) {
		sort.Sort(nl)
	}
}

func (nl *nickList) Remove(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < nl.Len() && nl.slice[i].name == n.name {
		nl.slice = append(nl.slice[0:i], nl.slice[i+1:]...)
	}
}

func (nl *nickList) Replace(old, new string) {
	n := splitNick(old)
	i := nl.FindIndex(n)
	if i < nl.Len() && nl.slice[i].name == n.name {
		a := nl.slice[i]
		b := splitNick(new)
		b.prefix = a.prefix
		log.Println("old:", a, "new:", b)
		nl.Remove(old)
		nl.Add(b.String())
	}
}

func (nl *nickList) GetPrefix(nick string) string {
	n := splitNick(nick)
	i := nl.FindIndex(n)
	if i < nl.Len() && nl.slice[i].name == n.name {
		return nl.slice[i].prefix
	}
	return ""
}

func (nl *nickList) SetPrefix(nick, prefix string) {
	n := splitNick(nick)
	i := nl.FindIndex(n)
	if i < nl.Len() && nl.slice[i].name == n.name {
		nl.slice[i].prefix = sortPrefix(prefix)
	}
}

func (nl *nickList) String() string {
	return fmt.Sprintf("%s", nl.slice)
}

func (nl *nickList) StringSlice() []string {
	s := []string{}
	var byprefix nickListByPrefix = nl.slice[:]
	sort.Sort(byprefix)
	for _, n := range byprefix {
		s = append(s, (*n).String())
	}
	return unique(s)
}

// NOTE: this is not really a good solution...

func unique(slice []string) []string {
	for i := len(slice) - 1; i >= 0; i-- {
		s := slice[i]
		j := sort.SearchStrings(slice, s)
		if i != j {
			slice = remove(slice, j)
		}
	}
	return slice
}

func remove(slice []string, i int) []string {
	return append(slice[0:i], slice[i+1:]...)
}
