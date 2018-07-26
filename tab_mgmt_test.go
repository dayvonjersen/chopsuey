package main

import (
	"sync"
	"testing"
)

func TestTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(5)
	var t0, t1, t2, t3, t4 *tabWithContext
	go func() {
		t0 = tabMan.CreateTab(&tabContext{}, 0)
		wg.Done()
	}()
	go func() {
		t1 = tabMan.CreateTab(&tabContext{}, 1)
		wg.Done()
	}()
	go func() {
		t2 = tabMan.CreateTab(&tabContext{}, 2)
		wg.Done()
	}()
	go func() {
		t3 = tabMan.CreateTab(&tabContext{}, 3)
		wg.Done()
	}()
	go func() {
		t4 = tabMan.CreateTab(&tabContext{}, 4)
		wg.Done()
	}()
	wg.Wait()

	finder := func(index int) func(t *tabWithContext) bool {
		return func(t *tabWithContext) bool {
			if t.tab.Index() == index {
				return true
			}
			return false
		}
	}

	if t0 != tabMan.FindTab(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.FindTab(finder(1)) {
		test.Fail()
	}
	if t2 != tabMan.FindTab(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.FindTab(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.FindTab(finder(4)) {
		test.Fail()
	}
}

func TestTabDelete(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(5)
	var t0, t1, t2, t3, t4 *tabWithContext
	go func() {
		t0 = tabMan.CreateTab(&tabContext{}, 0)
		wg.Done()
	}()
	go func() {
		t1 = tabMan.CreateTab(&tabContext{}, 1)
		wg.Done()
	}()
	go func() {
		t2 = tabMan.CreateTab(&tabContext{}, 2)
		wg.Done()
	}()
	go func() {
		t3 = tabMan.CreateTab(&tabContext{}, 3)
		wg.Done()
	}()
	go func() {
		t4 = tabMan.CreateTab(&tabContext{}, 4)
		wg.Done()
	}()
	wg.Wait()

	tabMan.DeleteTab(t2.tab)

	finder := func(index int) func(t *tabWithContext) bool {
		return func(t *tabWithContext) bool {
			if t.tab.Index() == index {
				return true
			}
			return false
		}
	}

	if t0 != tabMan.FindTab(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.FindTab(finder(1)) {
		test.Fail()
	}
	if nil != tabMan.FindTab(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.FindTab(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.FindTab(finder(4)) {
		test.Fail()
	}
}
