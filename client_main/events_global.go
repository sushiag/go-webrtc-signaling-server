package main

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

func captureGlobalEvents(gtx layout.Context, app *appState, sys system) {
	if app.focus.hasFocusedInput {
		return
	}

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, app)

	keyFilter := key.Filter{
		Focus:    nil,
		Required: 0,
		Optional: key.ModCtrl,
		Name:     "",
	}
	for {
		ev, ok := gtx.Event(keyFilter)
		if !ok {
			break
		}

		switch event := ev.(type) {
		case key.Event:
			// toggle color palette
			if event.Name == "P" && event.Modifiers&key.ModCtrl != 0 && event.State == key.Press {
				if g, _, ok := sys.tryGetGraphicsComponentRef(app.colorPalette); ok {
					g.isDisabled = !g.isDisabled
				}
			}
		}
	}
}
