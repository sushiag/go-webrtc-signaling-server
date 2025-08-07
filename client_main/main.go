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

	for {
		ev := window.Event()
		switch ev := ev.(type) {
		case app.DestroyEvent:
			return ev.Err

		case app.FrameEvent:
			{

				// reset the operations (required by gio)
				gtx := app.NewContext(&ops, ev)

				captureGlobalKeyEvents(gtx, appState, uiSystem)
				captureAndProcessEvents(
					gtx,
					appState,
					*uiSystem.interactables,
					*uiSystem.states,
					*uiSystem.graphics,
				)
				declareEventRegions(gtx, *uiSystem.interactables)
				drawGraphics(gtx, *uiSystem.graphics, uiSystem.textShaper)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func captureGlobalKeyEvents(gtx layout.Context, state *appState, ui uiSystem) {
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
			if g, idx, ok := ui.graphics.getComponent(state.colorPalette); ok {
				ui.graphics.components[idx].isDisabled = !g.isDisabled
			}
		}

	}
}

func captureAndProcessEvents(
	gtx layout.Context,
	appState *appState,
	interactables system[interactableComponent],
	states system[stateComponent],
	graphics system[graphicsComponent],
) {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	filters := make([]event.Filter, 0, 2)
	for idx, comp := range interactables.components {
		if comp.isDisabled {
			continue
		}
		entity := interactables.getEntity(idx)

		filters = comp.getEventFilters(filters)

		for {
			event, ok := gtx.Event(filters...)
			if !ok {
				break
			}

			processEvent(event, entity, appState, states, graphics)
		}

		filters = filters[:0]
	}
}

func declareEventRegions(gtx layout.Context, iteractables system[interactableComponent]) {
	for _, interactable := range iteractables.components {
		if interactable.isDisabled {
			continue
		}
		interactable.declareEventRegion(gtx)
	}
}

func processEvent(
	event event.Event,
	entity entity,
	appState *appState,
	states system[stateComponent],
	graphics system[graphicsComponent],
) {
	switch appState.currentPage {
	case apploginPage:
		if entity == appState.login.loginBtn {
			ptrEv, ok := event.(pointer.Event)
			if !ok {
				return
			}

			if s, idx, ok := states.getComponent(entity); ok {
				nextState := s.processBtnEvent(ptrEv.Kind)
				states.components[idx].state = nextState

				if g, idx, ok := graphics.getComponent(entity); ok {
					graphics.components[idx].bgColor = g.colors[nextState]
				}

				if ptrEv.Kind == pointer.Press {
					fmt.Println("login pressed")
				}
			}
		}
	case appMainPage:
	}
}

func drawGraphics(gtx layout.Context, graphics system[graphicsComponent], textShaper *text.Shaper) {
	defer clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Push(gtx.Ops).Pop()

	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorGray])

	for _, graphics := range graphics.components {
		if graphics.isDisabled {
			continue
		}
		graphics.draw(gtx, textShaper)
	}
}
