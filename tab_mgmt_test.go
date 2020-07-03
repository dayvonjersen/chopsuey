package main

import (
	"sync"
	"testing"
)

type dummyTab struct {
	index int
}

func (t *dummyTab) Index() int         { return t.index }
func (t *dummyTab) Title() string      { return "dummy tab" }
func (t *dummyTab) StatusIcon() string { return "" }
func (t *dummyTab) StatusText() string { return "" }
func (t *dummyTab) HasFocus() bool     { return false }
func (t *dummyTab) Focus()             {}
func (t *dummyTab) Close()             {}

func TestTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(5)
	var t0, t1, t2, t3, t4 *tabWithContext
	go func() {
		t0 = tabMan.Create(&tabContext{}, 0)
		t0.tab = &dummyTab{index: 0}
		wg.Done()
	}()
	go func() {
		t1 = tabMan.Create(&tabContext{}, 1)
		t1.tab = &dummyTab{index: 1}
		wg.Done()
	}()
	go func() {
		t2 = tabMan.Create(&tabContext{}, 2)
		t2.tab = &dummyTab{index: 2}
		wg.Done()
	}()
	go func() {
		t3 = tabMan.Create(&tabContext{}, 3)
		t3.tab = &dummyTab{index: 3}
		wg.Done()
	}()
	go func() {
		t4 = tabMan.Create(&tabContext{}, 4)
		t4.tab = &dummyTab{index: 4}
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

	if t0 != tabMan.Find(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.Find(finder(1)) {
		test.Fail()
	}
	if t2 != tabMan.Find(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.Find(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.Find(finder(4)) {
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
		t0 = tabMan.Create(&tabContext{}, 0)
		t0.tab = &dummyTab{index: 0}
		wg.Done()
	}()
	go func() {
		t1 = tabMan.Create(&tabContext{}, 1)
		t1.tab = &dummyTab{index: 1}
		wg.Done()
	}()
	go func() {
		t2 = tabMan.Create(&tabContext{}, 2)
		t2.tab = &dummyTab{index: 2}
		wg.Done()
	}()
	go func() {
		t3 = tabMan.Create(&tabContext{}, 3)
		t3.tab = &dummyTab{index: 3}
		wg.Done()
	}()
	go func() {
		t4 = tabMan.Create(&tabContext{}, 4)
		t4.tab = &dummyTab{index: 4}
		wg.Done()
	}()
	wg.Wait()

	tabMan.Delete(t2)

	finder := func(index int) func(t *tabWithContext) bool {
		return func(t *tabWithContext) bool {
			if t.tab.Index() == index {
				return true
			}
			return false
		}
	}

	if t0 != tabMan.Find(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.Find(finder(1)) {
		test.Fail()
	}
	if nil != tabMan.Find(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.Find(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.Find(finder(4)) {
		test.Fail()
	}
}

func TestNoDuplicateTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}

	race := func() {
		ctx := tabMan.CreateIfNotFound(&tabContext{}, 0xff, func(t *tabWithContext) bool {
			return t.tab.Index() == 0xff
		})
		if ctx.tab == nil {
			ctx.tab = &dummyTab{index: 0xff}
		}
		wg.Done()
	}

	horses := 1000
	wg.Add(horses)
	for i := 0; i < horses; i++ {
		go race()
	}
	wg.Wait()

	if len(tabMan.tabs) != 1 {
		printf(tabMan.tabs)
		test.Fail()
	}
}

func TestDoWeEvenNeedUpdate(test *testing.T) { // apparently we don't
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	servState := &serverState{connState: CONNECTED}
	{
		ctx := tabMan.Create(&tabContext{}, 0)
		ctx.servState = servState
	}

	{
		ctx := tabMan.Find(allTabsFinder)
		if ctx.servState != servState {
			test.Fail()
		}
	}

}
