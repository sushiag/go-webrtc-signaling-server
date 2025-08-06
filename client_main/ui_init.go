package main

import (
	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/text"
)

type uiSystem struct {
	states        map[entity]stateComponent
	interactables map[entity]interactableComponent
	graphics      map[entity]graphicsComponent
	textShaper    *text.Shaper
}

func initUIEntities(appState *appState) uiSystem {
	states := make(map[entity]stateComponent, 32)
	bounds := make(map[entity]interactableComponent, 32)
	graphics := make(map[entity]graphicsComponent, 32)
	textShaper := text.NewShaper(text.WithCollection(gofont.Collection()))
	uiSystem := uiSystem{
		states,
		bounds,
		graphics,
		textShaper,
	}

	appState.colorPalette = makeColorPalette(&uiSystem)

	appState.login.loginBtn = makeButton(&uiSystem, "test", colorPurpleDarker, colorPurpleLight, colorPurpleDark, colorWhite)

	return uiSystem
}

func makeColorPalette(ui *uiSystem) entity {
	e := newEntity()

	ui.graphics[e] = graphicsComponent{
		posX:       0,
		posY:       0,
		entityID:   e,
		entityKind: entityKindColorPalette,
	}

	return e
}

func makeButton(ui *uiSystem, txt string, colorDisabled, colorIdle, colorPressed, colorHovered colorID) entity {
	e := newEntity()

	ui.states[e] = stateComponent{entityID: e, kind: entityKindButton, state: 0}

	x := 100
	y := 100

	ui.interactables[e] = interactableComponent{
		posX:       x,
		posY:       y,
		entityID:   e,
		width:      100,
		height:     50,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}

	colors := [8]colorID{colorDisabled, colorIdle, colorPressed, colorHovered}
	ui.graphics[e] = graphicsComponent{
		posX:       x,
		posY:       y,
		entityID:   e,
		text:       txt,
		textColor:  colorWhite,
		colors:     colors,
		width:      100,
		height:     50,
		bgColor:    colors[btnStateIdle],
		entityKind: entityKindButton,
	}

	return e
}
