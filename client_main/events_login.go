package main

import (
	"gioui.org/io/event"
	"gioui.org/layout"
)

func processEvLoginPage(
	gtx layout.Context,
	app *appState,
	sys system,
	eventFilters *[]event.Filter,
) {
	for idx, interactable := range sys.interactables.comps {
		if interactable.isDisabled {
			continue
		}

		entity := sys.interactables.getEntity(idx)
		state := sys.getStateComponentRef(entity)

		switch state.bundleKind {
		case bundleButton:
			interactable.getEventFilters(eventFilters)
			for {
				ev, ok := gtx.Event(*eventFilters...)
				if !ok {
					break
				}

				graphics := sys.getGraphicsComponentRef(entity)
				updateBtnBundle(ev, state, graphics)

				if entity == app.login.loginBtn && state.state == btnStatePressed {
					sys.reset()
					app.currentPage = appMainPage
					initMainPageEntities(app, sys)
					return
				}
			}
		case bundleTextInput:
			interactable.getEventFilters(eventFilters)
			for {
				ev, ok := gtx.Event(*eventFilters...)
				if !ok {
					break
				}

				graphics := sys.getGraphicsComponentRef(entity)
				updateTextInputBundle(
					ev,
					entity,
					state,
					graphics,
					gtx,
					app,
				)
			}
		}

		*eventFilters = (*eventFilters)[:0]
	}
}
