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
	if !nl.Has("zebra") {
		t.Fatalf("has is broken")
	}
	nl.Add("@yak")
	if !nl.Has("@yak") {
		n := splitNick("@yak")
		fmt.Printf("%#v %#v %#v\n", nl, n, nl.FindIndex(n))
		t.Fatalf("has is broken")
	}
	nl.Add("+xenyx")
	if !nl.Has("+xenyx") {
		t.Fatalf("has is broken")
	}
	nl.Add("walrus")
	if !nl.Has("walrus") {
		t.Fatalf("has is broken")
	}
	nl.Add("%velociraptor")
	if !nl.Has("%velociraptor") {
		t.Fatalf("has is broken")
	}
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
