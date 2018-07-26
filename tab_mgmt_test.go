package main

import "testing"

func TestTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	t := tabMan.CreateTab(&tabContext{}, 0)
	if t.tab.Index() != 0 {
		test.Fail()
	}
}
