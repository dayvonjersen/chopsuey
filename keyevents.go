package main

import "github.com/lxn/walk"

func ctrlTab(key walk.Key) {
	if key == walk.KeyTab && walk.ControlDown() {
		max := tabWidget.Pages().Len() - 1
		if max < 1 {
			return
		}
		index := tabWidget.CurrentIndex()
		if walk.ShiftDown() {
			index -= 1
			if index < 0 {
				index = max
			}
		} else {
			index += 1
			if index > max {
				index = 0
			}
		}
		tabWidget.SetCurrentIndex(index)
	}
}
