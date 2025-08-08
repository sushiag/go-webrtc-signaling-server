package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	sys "gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
)

func main() {
	go func() {
		window := new(app.Window)
		window.Option(func(m unit.Metric, c *app.Config) {
			c.Title = "chatapp"
			c.Focused = true
		})
		window.Perform(sys.ActionRaise)

		appState := appState{}
		systems := systems{
			newSystem[stateComponent](),
			newSystem[boundingBoxComponent](),
			newSystem[interactableComponent](),
			newSystem[graphicsComponent](),
			text.NewShaper(text.WithCollection(gofont.Collection())),
		}

		initEntities(&appState, systems)

		handleWindowEvents(window, systems, &appState)

		os.Exit(0)
	}()
	app.Main()
}

// Starts a blocking loop that will handle window events
func handleWindowEvents(
	window *app.Window,
	systems systems,
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

				// layout step
				switch appState.currentPage {
				case apploginPage:
					layoutLoginPage(gtx, *systems.bBoxes, appState.login)
				case appMainPage:
				}

				captureGlobalKeyEvents(gtx, appState, systems)
				captureAndProcessEvents(
					gtx,
					appState,
					*systems.interactables,
					*systems.states,
					*systems.graphics,
				)
				declareEventRegions(gtx, *systems.bBoxes, *systems.interactables)
				drawGraphics(gtx, *systems.bBoxes, *systems.graphics, systems.textShaper)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func captureGlobalKeyEvents(gtx layout.Context, state *appState, ui systems) {
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

func declareEventRegions(gtx layout.Context, bBoxes system[boundingBoxComponent], iteractables system[interactableComponent]) {
	for idx, iComp := range iteractables.components {
		if iComp.isDisabled {
			continue
		}

		entity := iteractables.getEntity(idx)
		bb, _, ok := bBoxes.getComponent(entity)
		if !ok {
			log.Panicf("tried to declare an event region for '%d' without a bounding box", entity)
		}

		iComp.declareEventRegion(gtx, bb)
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
		for idx, s := range states.components {
			ptrEv, ok := event.(pointer.Event)
			if !ok {
				return
			}

			nextState := s.processBtnEvent(ptrEv.Kind)
			states.components[idx].state = nextState

			if g, idx, ok := graphics.getComponent(entity); ok {
				graphics.components[idx].bgColor = g.bgColors[nextState]
				graphics.components[idx].textColor = g.textColors[nextState]
			}
		}
	case appMainPage:
	}
}

func drawGraphics(
	gtx layout.Context,
	bBoxes system[boundingBoxComponent],
	graphics system[graphicsComponent],
	textShaper *text.Shaper,
) {
	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorGray])

	for idx, g := range graphics.components {
		if g.isDisabled {
			continue
		}

		entity := graphics.getEntity(idx)

		bb, _, ok := bBoxes.getComponent(entity)
		if !ok {
			log.Panicf("[ERR] tried to draw graphics for entity '%d' with no bounding box", entity)
		}

		g.draw(gtx, bb, textShaper)
	}
}
