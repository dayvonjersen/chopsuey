package main

import (
	"log"
	"regexp"
	"sort"
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

func sortPrefix(prefix string) string {
	s := []byte(prefix)
	sort.Slice(s, func(i, j int) bool {
		a, b := s[i], s[j]
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
	})
	return string(s)
}

type nickListByPrefix []*nick

func (nl nickListByPrefix) Len() int { return len(nl) }
func (nl nickListByPrefix) Less(i, j int) bool {
	if nl[i].prefix == nl[j].prefix {
		return nl[i].name < nl[j].name
	}
	if len(nl[i].prefix) == 0 {
		return false
	}
	if len(nl[j].prefix) == 0 {
		return true
	}
	a, b := nl[i].prefix[0], nl[j].prefix[0]
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
}
func (nl nickListByPrefix) Swap(i, j int) { nl[i], nl[j] = nl[j], nl[i] }

type nickList []*nick

func (nl nickList) Len() int           { return len(nl) }
func (nl nickList) Less(i, j int) bool { return nl[i].name < nl[j].name }
func (nl nickList) Swap(i, j int)      { nl[i], nl[j] = nl[j], nl[i] }

func (nl *nickList) FindIndex(n *nick) int {
	return nl.FindIndexSelection(n)
}

func (nl *nickList) FindIndexSelection(n *nick) int {
	for i, o := range *nl {
		if o.name == n.name {
			return i
		}
	}
	return len(*nl)
}

func (nl *nickList) FindIndexBinary(n *nick) int {
	i, j := 0, len(*nl)-1
	for i <= j {
		k := (i + j) / 2
		o := (*nl)[k].name
		if o > n.name {
			j = k - 1
		} else if o < n.name {
			i = k + 1
		} else {
			return k
		}
	}
	return len(*nl)
}

func (nl *nickList) Has(prefixed string) bool {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		return true
	}
	return false
}

func (nl *nickList) Add(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	n.prefix = sortPrefix(n.prefix)
	if i < len(*nl) && (*nl)[i].name == n.name {
		if (*nl)[i].prefix != n.prefix {
			(*nl)[i].prefix = n.prefix
		}
	} else {
		(*nl) = append((*nl)[:i], append([]*nick{n}, (*nl)[i:]...)...)
	}
	sort.Sort(*nl)
}

func (nl *nickList) Remove(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		(*nl) = append((*nl)[0:i], (*nl)[i+1:]...)
	}
	sort.Sort(*nl)
}

func (nl *nickList) Replace(old, new string) {
	n := splitNick(old)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		a := (*nl)[i]
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
	if i < len(*nl) && (*nl)[i].name == n.name {
		return (*nl)[i].prefix
	}
	return ""
}

func (nl *nickList) SetPrefix(nick, prefix string) {
	n := splitNick(nick)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		(*nl)[i].prefix = sortPrefix(prefix)
	}
}

func (nl *nickList) StringSlice() []string {
	s := []string{}
	byprefix := (*nickListByPrefix)(nl)
	sort.Sort(byprefix)
	for _, n := range *byprefix {
		s = append(s, n.String())
	}
	return s
}
