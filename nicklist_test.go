package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestNickListSync(t *testing.T) {
	f, err := os.Open("nicklist_test.data")
	checkErr(err)
	defer f.Close()

	nl := &nickList{}
	nl.Init()

	all := []string{}
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		nicks := strings.Split(scanner.Text(), " ")
		all = append(all, nicks...)
		for _, n := range nicks {
			nl.add <- n
			<-nl.updateRequest
		}
	}

	for _, n := range all {
		if !nl.Has(n) {
			t.Fatal(n, "was not Added to nickList")
		}
		nl.remove <- n
		<-nl.updateRequest
		if nl.Has(n) {
			t.Fatal(n, "was Added multiple times")
		}
	}
}

func TestNickListAsync(t *testing.T) {
	f, err := os.Open("nicklist_test.data")
	checkErr(err)
	defer f.Close()

	nl := &nickList{}
	nl.Init()

	all := []string{}
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	done := make(chan struct{})
	count := 0
	for scanner.Scan() {
		count++
		nicks := strings.Split(scanner.Text(), " ")
		all = append(all, nicks...)
		go func() {
			for _, n := range nicks {
				if n != "" {
					if nl.Has(n) {
						split := splitNick(n)
						nl.setPrefix <- []string{n, split.prefix}
					} else {
						nl.add <- n
					}
					<-nl.updateRequest
				}
			}
			done <- struct{}{}
		}()
	}

	i := 0
	for {
		<-done
		i++
		if i == count {
			close(done)
			break
		}
	}
	for _, n := range all {
		if !nl.Has(n) {
			t.Fatal(n, "was not Added to nickList")
		}
		nl.remove <- n
		<-nl.updateRequest
		if nl.Has(n) {
			t.Fatal(n, "was Added multiple times")
		}
	}
}

func TestNickList(t *testing.T) {
	nl := &nickList{}
	nl.Init()
	nl.add <- "something"
	<-nl.updateRequest
	if !nl.Has("something") {
		t.Fatalf("%#v", nl)
	}
	if nl.FindIndex(&nick{name: "something"}) != 0 {
		t.Fatal("nickList.FindIndex is broken")
	}
	if !nl.Has("something") {
		t.Fatal("nickList.Has is broken")
	}
	nl.remove <- "something"
	<-nl.updateRequest
	if nl.Has("something") {
		t.Fatalf("%#v", nl)
	}
	if nl.Has("something") {
		t.Fatalf("nickList.Has is broken")
	}

	nl.add <- "someone"
	<-nl.updateRequest
	nl.add <- "@someone"
	<-nl.updateRequest
	if nl.StringSlice()[0] != "@someone" {
		printf(nl)
		t.Fatalf("%#v", nl.StringSlice())
	}
	nl.add <- "someone"
	<-nl.updateRequest
	if nl.StringSlice()[0] != "someone" {
		t.Fatalf("%#v", nl)
	}
	nl.Shutdown()

	nl = &nickList{}
	nl.Init()
	nl.add <- "zebra"
	<-nl.updateRequest
	if !nl.Has("zebra") {
		t.Fatalf("has is broken")
	}
	nl.add <- "@yak"
	<-nl.updateRequest
	if !nl.Has("@yak") {
		n := splitNick("@yak")
		fmt.Printf("%#v %#v %#v\n", nl, n, nl.FindIndex(n))
		t.Fatalf("has is broken")
	}
	nl.add <- "+xenyx"
	<-nl.updateRequest
	if !nl.Has("+xenyx") {
		t.Fatalf("has is broken")
	}
	nl.add <- "walrus"
	<-nl.updateRequest
	if !nl.Has("walrus") {
		t.Fatalf("has is broken")
	}
	nl.add <- "%velociraptor"
	<-nl.updateRequest
	if !nl.Has("%velociraptor") {
		t.Fatalf("has is broken")
	}
	expect := "[%velociraptor walrus +xenyx @yak zebra]"
	actual := fmt.Sprintf("%v", nl)
	if expect != actual {
		t.Fatal("\nexpect:", expect, "\nactual:", actual)
	}
	expect = "[@yak %velociraptor +xenyx walrus zebra]"
	actual = fmt.Sprintf("%v", nl.StringSlice())
	if expect != actual {
		t.Fatal("\nexpect:", expect, "\nactual:", actual)
	}
}
