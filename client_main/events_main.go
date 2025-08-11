package main

import (
	"gioui.org/io/event"
	"gioui.org/layout"
)

func processEvMainPage(
	gtx layout.Context,
	app *appState,
	sys system,
	eventFilters *[]event.Filter,
) {
	for idx, iComp := range sys.interactables.comps {
		if iComp.isDisabled {
			continue
		}

		entity := sys.interactables.getEntity(idx)
		state := sys.getStateComponentRef(entity)

		switch state.bundleKind {
		case bundleButton:
			iComp.getEventFilters(eventFilters)
			for {
				ev, ok := gtx.Event(*eventFilters...)
				if !ok {
					break
				}

				graphics := sys.getGraphicsComponentRef(entity)
				updateBtnBundle(ev, state, graphics)
			}
		}

		*eventFilters = (*eventFilters)[:0]
	}
}
