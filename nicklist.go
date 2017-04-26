package main

import (
	"regexp"
	"sort"
)

type nick struct {
	prefix, name string
}

func (n *nick) String() string {
	return n.prefix + n.name
}

var nickRegex = regexp.MustCompile("^([@+]*)(.+)$")

func splitNick(prefixed string) *nick {
	m := nickRegex.FindAllStringSubmatch(prefixed, -1)
	return &nick{m[0][1], m[0][2]}
}

type nickList []*nick

// sort.Interface
func (nl nickList) Len() int           { return len(nl) }
func (nl nickList) Less(i, j int) bool { return nl[i].name < nl[j].name }
func (nl nickList) Swap(i, j int)      { nl[i], nl[j] = nl[j], nl[i] }

func (nl *nickList) FindIndex(n *nick) int {
	return sort.Search(len(*nl), func(i int) bool {
		return (*nl)[i].name == n.name
	})
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
	if i < len(*nl) && (*nl)[i].name == n.name {
		if (*nl)[i].prefix != n.prefix {
			(*nl)[i].prefix = n.prefix
		}
	} else {
		(*nl) = append(*nl, n)
	}
}

func (nl *nickList) Remove(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		(*nl) = append((*nl)[0:i], (*nl)[i+1:]...)
	}
}

func (nl nickList) StringSlice() []string {
	s := []string{}
	for _, n := range nl {
		s = append(s, n.String())
	}
	return s
}
