package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	gioSys "gioui.org/io/system"
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
		window.Perform(gioSys.ActionRaise)

		appState := appState{}
		system := newSystem()
		textShaper := text.NewShaper(text.WithCollection(gofont.Collection()))
		appState.colorPalette = makeColorPalette(system)

		initLoginPageEntities(&appState, system)

		handleWindowEvents(window, system, textShaper, &appState)

		os.Exit(0)
	}()
	app.Main()
}

// Starts a blocking loop that will handle window events
func handleWindowEvents(
	window *app.Window,
	sys system,
	textShaper *text.Shaper,
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
					layoutLoginPage(gtx, sys, appState.login)
				case appMainPage:
					layoutMainPage(gtx, sys, appState.main)
				}

				captureGlobalEvents(gtx, appState, sys)
				captureAndProcessEvents(
					gtx,
					appState,
					sys,
				)
				declareEventRegions(gtx, sys)
				drawGraphics(gtx, sys, textShaper)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func captureAndProcessEvents(
	gtx layout.Context,
	appState *appState,
	sys system,
) {
	interactables := *sys.interactables

	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	filters := make([]event.Filter, 0, 4)
	for idx, iComp := range interactables.comps {
		if iComp.isDisabled {
			continue
		}
		entity := sys.interactables.getEntity(idx)

		filters = iComp.getEventFilters(filters)

		for {
			ev, ok := gtx.Event(filters...)
			if !ok {
				break
			}

			switch appState.currentPage {
			case apploginPage:
				pageChanged := processEvLoginPage(gtx, appState, ev, sys, entity)
				if pageChanged {
					return
				}
			case appMainPage:
				processEvMainPage(gtx, appState, ev, sys, entity)
			default:
				log.Println("[WARN] no event handler set for the current page:", appState.currentPage)
			}
		}

		filters = filters[:0]
	}
}

func declareEventRegions(gtx layout.Context, sys system) {
	for idx, iComp := range sys.interactables.comps {
		if iComp.isDisabled {
			continue
		}

		entity := sys.interactables.getEntity(idx)
		bboxComp := sys.getBBoxComponent(entity)

		iComp.declareEventRegion(gtx, bboxComp)
	}
}

func drawGraphics(
	gtx layout.Context,
	sys system,
	textShaper *text.Shaper,
) {
	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorPurpleDark])

	for idx, g := range sys.graphics.comps {
		if g.isDisabled {
			continue
		}

		entity := sys.graphics.getEntity(idx)

		bbox := sys.getBBoxComponent(entity)

		g.draw(gtx, bbox, textShaper)
	}
}
