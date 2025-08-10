package main

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

func processEvMainPage(
	gtx layout.Context,
	app *appState,
	ev event.Event,
	sys system,
	e entity,
) bool {
	stateComp := sys.getStateComponentRef(e)
	graphicsComp, _, hasGraphics := sys.tryGetGraphicsComponentRef(e)

	switch event := ev.(type) {
	case pointer.Event:
		stateComp.handlePtrInteraction(event.Kind)

		if hasGraphics {
			graphicsComp.bgColor = graphicsComp.bgColors[stateComp.state]
			graphicsComp.textColor = graphicsComp.textColors[stateComp.state]
		}
	}

	return false
}
