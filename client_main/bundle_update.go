package main

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

func updateBtnBundle(event event.Event, state *stateComponent, graphics *graphicsComponent) {
	ptrEv, ok := event.(pointer.Event)
	if !ok {
		return
	}

	state.state = getNextBtnState(state.state, ptrEv)

	graphics.bgColor = graphics.bgColors[state.state]
	graphics.textColor = graphics.textColors[state.state]
}

func updateTextInputBundle(
	ev event.Event,
	entity entity,
	state *stateComponent,
	graphicsComp *graphicsComponent,
	gtx layout.Context,
	app *appState,
) {
	switch event := ev.(type) {
	case pointer.Event:
		{
			state.state = getNextTxtInputState(state.state, event)

			graphicsComp.bgColor = graphicsComp.bgColors[state.state]
			graphicsComp.textColor = graphicsComp.textColors[state.state]

			if state.state == txtInputStateFocused {
				// move the focus onto this entity
				gtx.Execute(key.FocusCmd{Tag: entity})
				app.focus.hasFocusedInput = true
				app.focus.focusedEntity = entity
			}
		}

	case key.FocusEvent:
		{
			if event.Focus {
				// on focus
				state.state = txtInputStateFocused

				graphicsComp.bgColor = graphicsComp.bgColors[state.state]
				graphicsComp.textColor = graphicsComp.textColors[state.state]
				if graphicsComp.text == graphicsComp.placeholderText {
					graphicsComp.text = ""
				}

				app.focus.hasFocusedInput = true
			} else {
				// on unfocus
				state.state = txtInputStateIdle

				graphicsComp.bgColor = graphicsComp.bgColors[state.state]
				graphicsComp.textColor = graphicsComp.textColors[state.state]
				if graphicsComp.text == "" {
					graphicsComp.text = graphicsComp.placeholderText
				}
			}
		}

	case key.EditEvent:
		{
			graphicsComp.text += event.Text
		}

	case key.Event:
		{
			switch event.Name {
			case key.NameDeleteBackward:
				{
					l := len(graphicsComp.text)
					if l > 0 {
						graphicsComp.text = graphicsComp.text[:l-1]
					}
				}
			case key.NameEscape:
				{
					gtx.Execute(key.FocusCmd{Tag: nil})
					app.focus.hasFocusedInput = false

					// NOTE: 1 should be the idle state for all bundles
					state.state = 1

					graphicsComp.bgColor = graphicsComp.bgColors[1]
					graphicsComp.textColor = graphicsComp.textColors[1]

					if graphicsComp.text == "" {
						graphicsComp.text = graphicsComp.placeholderText
					}
				}
			}
		}
	}
}
