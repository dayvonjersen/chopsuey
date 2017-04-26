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

var nickRegex = regexp.MustCompile("^([@%+]*)(.+)$")

func splitNick(prefixed string) *nick {
	m := nickRegex.FindAllStringSubmatch(prefixed, -1)
	return &nick{m[0][1], m[0][2]}
}

type nickListByPrefix []*nick

func (nl nickListByPrefix) Len() int { return len(nl) }
func (nl nickListByPrefix) Less(i, j int) bool {
	a, b := nl[i].prefix, nl[j].prefix
	if a == b {
		return nl[i].name < nl[j].name
	}
	switch a {
	case "@":
		return true
	case "%":
		return b != "@"
	case "+":
		return b == ""
	case "":
		return false
	}

	panic("unhandled prefix: " + a)
}
func (nl nickListByPrefix) Swap(i, j int) { nl[i], nl[j] = nl[j], nl[i] }

type nickList []*nick

func (nl nickList) Len() int           { return len(nl) }
func (nl nickList) Less(i, j int) bool { return nl[i].name < nl[j].name }
func (nl nickList) Swap(i, j int)      { nl[i], nl[j] = nl[j], nl[i] }

func (nl *nickList) FindIndex(n *nick) int {
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
	if i < len(*nl) && (*nl)[i].name == n.name {
		if (*nl)[i].prefix != n.prefix {
			(*nl)[i].prefix = n.prefix
		}
	} else {
		(*nl) = append(*nl, n)
	}
	if !sort.IsSorted(*nl) {
		sort.Sort(*nl)
	}
}

func (nl *nickList) Remove(prefixed string) {
	n := splitNick(prefixed)
	i := nl.FindIndex(n)
	if i < len(*nl) && (*nl)[i].name == n.name {
		(*nl) = append((*nl)[0:i], (*nl)[i+1:]...)
	}
	if !sort.IsSorted(*nl) {
		sort.Sort(*nl)
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
