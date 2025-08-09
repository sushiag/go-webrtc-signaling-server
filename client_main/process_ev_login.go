package main

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

func processEvLoginPage(
	gtx layout.Context,
	app *appState,
	filters []event.Filter,
	sys system,
	stateComp *stateComponent,
	e entity,
) {
	for {
		event, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		ptrEv, ok := event.(pointer.Event)
		if !ok {
			return
		}

		nextState := stateComp.ptrInteraction(ptrEv.Kind)
		stateComp.state = nextState

		graphicsComp, ok := sys.tryGetGraphicsComponentRef(e)
		if ok {
			graphicsComp.bgColor = graphicsComp.bgColors[nextState]
			graphicsComp.textColor = graphicsComp.textColors[nextState]
		}
		if stateComp.kind == bundleTextInput && nextState == txtInputStateFocused {
			app.focusedInput = e
			app.hasFocusedInput = true
			gtx.Execute(key.FocusCmd{Tag: e})
			if ok && graphicsComp.text == graphicsComp.placeholderText {
				graphicsComp.text = ""
			}
		}
	}
}
