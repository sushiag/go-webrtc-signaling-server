package main

import (
	"fmt"
	"image"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

func main() {
	go func() {
		window := new(app.Window)

		ticker := time.NewTicker(time.Second)
		ticker.Stop()

		appState := appState{}
		uiSystem := initUIEntities(&appState)

		handleWindowEvents(window, uiSystem, &appState)

		os.Exit(0)
	}()
	app.Main()
}

// Starts a blocking loop that will handle window events
func handleWindowEvents(
	window *app.Window,
	uiSystem uiSystem,
	appState *appState,
) error {
	var ops op.Ops

	uiEvents := make([]interactionEvent, 0, 8)

	for {
		ev := window.Event()
		switch ev := ev.(type) {
		case app.DestroyEvent:
			return ev.Err

		case app.FrameEvent:
			{

				// reset the operations (required by gio)
				gtx := app.NewContext(&ops, ev)

				captureGlobalKeyEvents(gtx, appState, &uiSystem)
				captureComponentEvents(gtx, uiSystem.interactables, &uiEvents)
				processInteractionEvents(appState, &uiEvents, uiSystem.states, uiSystem.graphics)
				drawGraphics(gtx, uiSystem.graphics, uiSystem.textShaper)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func captureGlobalKeyEvents(gtx layout.Context, state *appState, ui *uiSystem) {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, state)

	for {
		// toggle color palette
		keyFilter := key.Filter{
			Focus:    nil,
			Required: key.ModCtrl,
			Optional: 0,
			Name:     "P",
		}
		ev, ok := gtx.Event(keyFilter)
		if !ok {
			break
		}

		keyEv, ok := ev.(key.Event)
		if !ok {
			return
		}

		if keyEv.State == key.Press {
			if colorPalette, ok := ui.graphics[state.colorPalette]; ok {
				colorPalette.isDisabled = !colorPalette.isDisabled
				ui.graphics[state.colorPalette] = colorPalette
			}
		}

	}
}

func captureComponentEvents(gtx layout.Context, iteractables map[entity]interactableComponent, outEvents *[]interactionEvent) {
	for _, interactable := range iteractables {
		if interactable.isDisabled {
			continue
		}
		interactable.captureEvents(gtx, outEvents)
	}
}

func processInteractionEvents(
	appState *appState,
	outEvents *[]interactionEvent,
	stateComps map[entity]stateComponent,
	graphicsComps map[entity]graphicsComponent,
) {
	switch appState.currentPage {
	case apploginPage:
		for _, ev := range *outEvents {
			if ev.entityID == appState.login.loginBtn {
				ptrEv, ok := ev.kind.(pointer.Event)
				if !ok {
					continue
				}

				if state, ok := stateComps[ev.entityID]; ok {
					newState := state.processBtnEvent(ptrEv.Kind)

					comp := graphicsComps[ev.entityID]
					comp.bgColor = comp.colors[newState]
					graphicsComps[ev.entityID] = comp

					if ptrEv.Kind == pointer.Press {
						fmt.Println("login pressed")
					}
				}

			}
		}
	case appMainPage:
	}

	// clear events
	*outEvents = (*outEvents)[:0]
}

func drawGraphics(gtx layout.Context, graphics map[entity]graphicsComponent, textShaper *text.Shaper) {
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()

	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorGray])

	for _, graphic := range graphics {
		if graphic.isDisabled {
			continue
		}
		graphic.draw(gtx, textShaper)
	}
}
