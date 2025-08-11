package main

import (
	"os"

	gioApp "gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	gioSys "gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
)

var globalTextShaper *text.Shaper

func main() {
	go func() {
		window := new(gioApp.Window)
		window.Option(func(m unit.Metric, c *gioApp.Config) {
			c.Title = "chatapp"
			c.Focused = true
		})
		window.Perform(gioSys.ActionRaise)

		appState := appState{}
		system := newSystem()
		globalTextShaper = text.NewShaper(text.WithCollection(gofont.Collection()))
		appState.colorPalette = makeColorPalette(system)

		initLoginPageEntities(&appState, system)

		handleWindowEvents(window, system, &appState)

		os.Exit(0)
	}()
	gioApp.Main()
}

// Starts a blocking loop that will handle window events
func handleWindowEvents(
	window *gioApp.Window,
	sys system,
	app *appState,
) error {
	var ops op.Ops

	// layout functions
	layoutPage := [_nAppPages]func(layout.Context, *appState, system){
		layoutLoginPage,
		layoutMainPage,
	}

	// event processing functions
	var eventFilters = make([]event.Filter, 0, 4)
	var processEvent = [_nAppPages]func(layout.Context, *appState, system, *[]event.Filter){
		processEvLoginPage,
		processEvMainPage,
	}

	for {
		ev := window.Event()
		switch ev := ev.(type) {
		case gioApp.DestroyEvent:
			return ev.Err

		case gioApp.FrameEvent:
			{
				// reset the operations (required by gio)
				gtx := gioApp.NewContext(&ops, ev)

				layoutPage[app.currentPage](gtx, app, sys)

				processEvent[app.currentPage](gtx, app, sys, &eventFilters)

				captureGlobalEvents(gtx, app, sys)

				drawGraphics(gtx, sys)

				declareEventRegions(gtx, sys)

				// update the display
				ev.Frame(gtx.Ops)
			}
		}
	}
}

func declareEventRegions(gtx layout.Context, sys system) {
	for idx, iteractable := range sys.interactables.comps {
		if iteractable.isDisabled {
			continue
		}

		entity := sys.interactables.getEntity(idx)
		bbox := sys.getBBoxComponent(entity)

		iteractable.declareEventRegion(gtx, bbox)
	}
}

func drawGraphics(
	gtx layout.Context,
	sys system,
) {
	// fill background color
	paint.Fill(gtx.Ops, colorPalette[colorPurpleDark])

	for idx, graphicsComp := range sys.graphics.comps {
		if graphicsComp.isDisabled {
			continue
		}

		entity := sys.graphics.getEntity(idx)

		bbox := sys.getBBoxComponent(entity)

		graphicsComp.draw(gtx, bbox, globalTextShaper)
	}
}
