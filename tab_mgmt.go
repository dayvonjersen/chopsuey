package main

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

type tabRequestCreate struct {
	ctx   *tabContext
	index int
	ret   chan *tabWithContext
}

type tabRequestSearch struct {
	finder func(*tabWithContext) bool
	ret    chan *tabWithContext
}

type tabRequestUpdate struct {
	// stub
}

type tabRequestDelete struct {
	tabs []tab
}

var tabMan = newTabManager()

type tabManager struct {
	tabs []*tabWithContext

	create chan *tabRequestCreate
	search chan *tabRequestSearch
	update chan *tabRequestUpdate
	delete chan *tabRequestDelete

	destroy chan struct{}
}

func (tabMan *tabManager) Shutdown() {
	close(tabMan.destroy)
}

func (tabMan *tabManager) CreateTab(ctx *tabContext, index int) *tabWithContext {
	ret := make(chan *tabWithContext)
	tabMan.create <- &tabRequestCreate{ctx, index, ret}
	return <-ret
}

func (tabMan *tabManager) FindTab(finder func(*tabWithContext) bool) *tabWithContext {
	ret := make(chan *tabWithContext)
	tabMan.search <- &tabRequestSearch{finder, ret}
	return <-ret
}

//
// finder funcs
// usage: tabMan.FindTabs(currentTabFinder)
//        tabMan.FindTabs(serverTabFinder(servState))
//        tabMan.FindTabs(channelTabFinder(chanState))
//        tabMan.FindTabs(someotherTabFinder) ...
//
func currentTabFinder(t *tabWithContext) bool {
	return t.tab.Index() == tabWidget.CurrentIndex()
}

func serverTabFinder(servState *serverState) func(*tabWithContext) bool {
	return func(t *tabWithContext) bool {
		if t.servState == servState {
			if _, ok := t.tab.(*tabServer); ok {
				return true
			}
		}
		return false
	}
}

func (tabMan *tabManager) DeleteTab(tabs ...tab) {
	tabMan.delete <- &tabRequestDelete{tabs}
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
		search:  make(chan *tabRequestSearch),
		update:  make(chan *tabRequestUpdate),
		delete:  make(chan *tabRequestDelete),
		destroy: make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-tabMan.destroy:
				return

			case req := <-tabMan.create:
				t := &tabWithContext{
					tab: &dummyTab{index: req.index},
				}
				t.servConn = req.ctx.servConn
				t.servState = req.ctx.servState
				t.chanState = req.ctx.chanState
				t.pmState = req.ctx.pmState

				tabMan.tabs = append(tabMan.tabs, t)
				req.ret <- t

				// switch on request type
				//      create tab of type
				// t = new tab
				// tabMan.tabs = append(tabMan.tabs, &tabWithContext{ctx, t}

			case req := <-tabMan.search:
				for _, t := range tabMan.tabs {
					if req.finder(t) {
						req.ret <- t
						break
					}
				}
				req.ret <- nil

			case req := <-tabMan.update:
				_ = req
				// stub

			case req := <-tabMan.delete:
				for _, t := range req.tabs {
					_ = t
					// delete(tabMan.tabs, t)
					// t.Close()
				}
			}
		}
	}()

	return tabMan
}
