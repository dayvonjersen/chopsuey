package main

import "testing"

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
		t.Fatal("nickList.Has is broken")
	}

	nl.Add("someone")
	nl.Add("@someone")
	if nl.StringSlice()[0] != "@someone" {
		t.Fatal("%#v", nl)
	}
	nl.Add("someone")
	if nl.StringSlice()[0] != "someone" {
		t.Fatal("%#v", nl)
	}
}
