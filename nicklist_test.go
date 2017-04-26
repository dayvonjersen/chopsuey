package main

import (
	"fmt"
	"testing"
)

func TestNickList(t *testing.T) {
	nl := &nickList{}
	nl.Add("something")
	if !nl.Has("something") {
		t.Fatalf("%#v", nl)
	}
	if nl.FindIndex(&nick{name: "something"}) != 0 {
		t.Fatal("nickList.FindIndex is broken")
	}
	if !nl.Has("something") {
		t.Fatal("nickList.Has is broken")
	}
	nl.Remove("something")
	if nl.Has("something") {
		t.Fatalf("%#v", nl)
	}
	if nl.Has("something") {
		t.Fatalf("nickList.Has is broken")
	}

	nl.Add("someone")
	nl.Add("@someone")
	if nl.StringSlice()[0] != "@someone" {
		t.Fatalf("%#v", nl)
	}
	nl.Add("someone")
	if nl.StringSlice()[0] != "someone" {
		t.Fatalf("%#v", nl)
	}

	nl = &nickList{}
	nl.Add("zebra")
	nl.Add("@yak")
	nl.Add("+xenyx")
	nl.Add("walrus")
	nl.Add("%velociraptor")
	expect := "&[%velociraptor walrus +xenyx @yak zebra]"
	actual := fmt.Sprintf("%v", nl)
	if expect != actual {
		t.Fatal("expect:", expect, "actual:", actual)
	}
	expect = "[@yak %velociraptor +xenyx walrus zebra]"
	actual = fmt.Sprintf("%v", nl.StringSlice())
	if expect != actual {
		t.Fatal("expect:", expect, "actual:", actual)
	}
}
