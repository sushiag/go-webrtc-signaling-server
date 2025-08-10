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

		if graphicsComp.kind == bundleButton {
			if e == app.login.loginBtn && stateComp.state == btnStatePressed {
				sys.reset()
				app.currentPage = appMainPage
				initMainPageEntities(app, sys)
				return true
			}
		}

		if hasGraphics && graphicsComp.kind == bundleTextInput {
			// on focus
			if stateComp.state != txtInputStateFocused {
				break
			}

			// remove and update the last focused
			if app.focus.hasFocusedInput && app.focus.focusedEntity != e {
				lastFocused := app.focus.focusedEntity
				gtx.Execute(key.FocusCmd{Tag: lastFocused})
				stateComp := sys.getStateComponentRef(lastFocused)
				stateComp.state = txtInputStateIdle
				if lastGraphicsComp, _, ok := sys.tryGetGraphicsComponentRef(lastFocused); ok {
					lastGraphicsComp.bgColor = graphicsComp.bgColors[stateComp.state]
					lastGraphicsComp.textColor = graphicsComp.textColors[stateComp.state]

					if lastGraphicsComp.text == "" {
						lastGraphicsComp.text = lastGraphicsComp.placeholderText
					}
				}
			}

			// update the current focused
			app.focus.focusedEntity = e
			app.focus.hasFocusedInput = true

			if graphicsComp.text == graphicsComp.placeholderText {
				graphicsComp.text = ""
			}

			gtx.Execute(key.FocusCmd{Tag: e})
		}

	case key.EditEvent:
		if app.focus.hasFocusedInput && app.focus.focusedEntity == e && hasGraphics {
			graphicsComp.text += event.Text
		}

	case key.Event:
		if app.focus.focusedEntity != e || !app.focus.hasFocusedInput {
			break
		}

		switch event.Name {
		case key.NameDeleteBackward:
			if hasGraphics {
				l := len(graphicsComp.text)
				if l > 0 {
					graphicsComp.text = graphicsComp.text[:l-1]
				}
			}
		case key.NameEscape:
			gtx.Execute(key.FocusCmd{Tag: e})
			app.focus.hasFocusedInput = false
			stateComp.state = 1

			if hasGraphics {
				graphicsComp.bgColor = graphicsComp.bgColors[1]
				graphicsComp.textColor = graphicsComp.textColors[1]

				if graphicsComp.text == "" {
					graphicsComp.text = graphicsComp.placeholderText
				}
			}
		}
	}

	return false
}
