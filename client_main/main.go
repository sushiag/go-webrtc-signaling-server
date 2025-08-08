package main

import (
	"fmt"
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
					systems,
				)
				declareEventRegions(gtx, *systems.bBoxes, *systems.interactables)
				drawGraphics(gtx, *systems.bBoxes, *systems.graphics, systems.textShaper)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func captureGlobalKeyEvents(gtx layout.Context, appState *appState, ui systems) {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, appState)

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

		keyEv, ok := ev.(key.Event)
		if !ok {
			return
		}

		if keyEv.State == key.Press && keyEv.Modifiers&key.ModCtrl != 0 {
			// toggle color palette
			if g, ok := ui.graphics.getComponentRef(appState.colorPalette); ok {
				g.isDisabled = !g.isDisabled
			}
		}

	}

	if appState.hasFocusedInput {
		for {
			ev, ok := gtx.Event(key.FocusFilter{Target: appState.focusedInput})
			if !ok {
				break
			}

			editEv, ok := ev.(key.EditEvent)
			if !ok {
				return
			}

			if g, ok := ui.graphics.getComponentRef(appState.focusedInput); ok {
				g.text += editEv.Text
			}
		}
	}
}

func captureAndProcessEvents(
	gtx layout.Context,
	appState *appState,
	systems systems,
) {
	interactables := *systems.interactables
	states := *systems.states
	graphics := *systems.graphics

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	filters := make([]event.Filter, 0, 2)
	for idx, comp := range interactables.components {
		if comp.isDisabled {
			continue
		}
		entity := interactables.getEntity(idx)
		stateComponent, ok := states.getComponentRef(entity)
		if !ok {
			log.Panicln("an interactable entity is missing a state component:", entity)
		}

		filters = comp.getEventFilters(filters)

		for {
			event, ok := gtx.Event(filters...)
			if !ok {
				break
			}

			ptrEv, ok := event.(pointer.Event)
			if !ok {
				return
			}

			switch appState.currentPage {
			case apploginPage:

				nextState := stateComponent.ptrInteraction(ptrEv.Kind)
				stateComponent.state = nextState

				g, ok := graphics.getComponentRef(entity)
				if ok {
					g.bgColor = g.bgColors[nextState]
					g.textColor = g.textColors[nextState]
				}
				if stateComponent.kind == bundleTextInput && nextState == txtInputStateFocused {
					appState.focusedInput = entity
					appState.hasFocusedInput = true
					gtx.Execute(key.FocusCmd{Tag: entity})
					fmt.Println("focused")
					if ok && g.text == g.placeholderText {
						g.text = ""
					}
				}
			case appMainPage:
				{
				}
			}
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

func drawGraphics(
	gtx layout.Context,
	bBoxes system[boundingBoxComponent],
	graphics system[graphicsComponent],
	textShaper *text.Shaper,
) {
	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorPurpleDark])

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
