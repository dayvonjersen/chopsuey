package main

import "log"

type tabWithContext struct {
	tabContext
	tab tab
}
type tabContext struct {
	servConn  *serverConnection
	servState *serverState

	chanState *channelState
	pmState   *privmsgState
}

type finderFunc func(*tabWithContext) bool

type tabRequestCreate struct {
	ctx    *tabContext
	index  int
	finder finderFunc
	ret    chan *tabWithContext
}

type tabRequestCount struct {
	ret chan int
}

type tabRequestSearch struct {
	finder finderFunc
	ret    chan []*tabWithContext
}

type tabRequestUpdate struct {
	// stub
}

type tabRequestDelete struct {
	tabs []tab
	ret  chan struct{}
}

type tabManager struct {
	tabs []*tabWithContext

	create chan *tabRequestCreate
	count  chan *tabRequestCount
	search chan *tabRequestSearch
	update chan *tabRequestUpdate
	delete chan *tabRequestDelete

	destroy chan struct{}
}

func (tabMan *tabManager) Shutdown() {
	close(tabMan.destroy)
}

func (tabMan *tabManager) Create(ctx *tabContext, index int) *tabWithContext {
	ret := make(chan *tabWithContext)
	go func() {
		tabMan.create <- &tabRequestCreate{ctx, index, nil, ret}
	}()
	return <-ret
}

func (tabMan *tabManager) CreateIfNotFound(ctx *tabContext, index int, finder finderFunc) *tabWithContext {
	ret := make(chan *tabWithContext)
	tabMan.create <- &tabRequestCreate{ctx, index, finder, ret}
	return <-ret
}

func (tabMan *tabManager) Len() int {
	ret := make(chan int)
	tabMan.count <- &tabRequestCount{ret}
	return <-ret
}

func (tabMan *tabManager) Find(finder finderFunc) *tabWithContext {
	ret := tabMan.FindAll(finder)
	if len(ret) > 0 {
		return ret[0]
	}
	return nil
}

func (tabMan *tabManager) FindAll(finder finderFunc) []*tabWithContext {
	ret := make(chan []*tabWithContext)
	tabMan.search <- &tabRequestSearch{finder, ret}
	return <-ret
}

//
// finder funcs
// usage: tabMan.Find(currentTabFinder)
//        tabMan.Find(serverTabFinder(servState))
//        tabMan.Find(channelTabFinder(chanState))
//        tabMan.Find(someotherTabFinder) ...
//
func allTabsFinder(t *tabWithContext) bool {
	return true
}

func currentTabFinder(t *tabWithContext) bool {
	return t.tab.Index() == tabWidget.CurrentIndex()
}

func serverTabFinder(servState *serverState) finderFunc {
	return func(t *tabWithContext) bool {
		if t.servState == servState {
			if _, ok := t.tab.(*tabServer); ok {
				return true
			}
		}
		return false
	}
}

func (tabMan *tabManager) Delete(tabs ...tab) {
	ret := make(chan struct{})
	tabMan.delete <- &tabRequestDelete{tabs, ret}
	<-ret
	return
}

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

func newTabManager() *tabManager {
	tabMan := &tabManager{
		tabs:    []*tabWithContext{},
		create:  make(chan *tabRequestCreate),
		count:   make(chan *tabRequestCount),
		search:  make(chan *tabRequestSearch),
		update:  make(chan *tabRequestUpdate),
		delete:  make(chan *tabRequestDelete),
		destroy: make(chan struct{}),
	}

	go func() {
		for {
		here:
			select {
			case <-tabMan.destroy:
				return

			case req := <-tabMan.create:
				log.Printf("got create: %v", req)
				if req.finder != nil {
					for _, t := range tabMan.tabs {
						if req.finder(t) {
							req.ret <- t
							break here
						}
					}
				}

				t := &tabWithContext{}
				t.servConn = req.ctx.servConn
				t.servState = req.ctx.servState
				t.chanState = req.ctx.chanState
				t.pmState = req.ctx.pmState

				tabMan.tabs = append(tabMan.tabs, t)
				req.ret <- t
				log.Printf("created sent return value to caller")

			case req := <-tabMan.count:
				log.Printf("got count: %v", req)
				req.ret <- len(tabMan.tabs)

			case req := <-tabMan.search:
				log.Printf("got search: %v", req)
				ret := []*tabWithContext{}
				for _, t := range tabMan.tabs {
					if req.finder(t) {
						ret = append(ret, t)
					}
				}
				req.ret <- ret

			case req := <-tabMan.update:
				log.Printf("got update: %v", req)
				_ = req
				// stub

			case req := <-tabMan.delete:
				log.Printf("got delete: %v", req)
				indices := []int{}
				for _, t := range req.tabs {
					indices = append(indices, t.Index())
				}

				for i, t := range tabMan.tabs {
					for _, index := range indices {
						if t.tab.Index() == index {
							tabMan.tabs = append(tabMan.tabs[0:i], tabMan.tabs[i+1:]...)
						}
					}

				}
				req.ret <- struct{}{}

				// t.Close()
			}
		}
	}()

	return tabMan
}
