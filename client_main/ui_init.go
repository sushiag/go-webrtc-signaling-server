package main

import (
	"fmt"

	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/text"
)

type uiSystem struct {
	states        *system[stateComponent]
	interactables *system[interactableComponent]
	graphics      *system[graphicsComponent]
	textShaper    *text.Shaper
}

func initUIEntities(appState *appState) uiSystem {
	textShaper := text.NewShaper(text.WithCollection(gofont.Collection()))
	uiSystem := uiSystem{
		newSystem[stateComponent](),
		newSystem[interactableComponent](),
		newSystem[graphicsComponent](),
		textShaper,
	}

	appState.colorPalette = makeColorPalette(uiSystem)
	appState.login.loginBtn = makeButton(uiSystem, "test", colorPurpleDarker, colorPurpleLight, colorPurpleDark, colorWhite)

	fmt.Println(len(uiSystem.graphics.components))

	return uiSystem
}

func makeColorPalette(ui uiSystem) entity {
	e := newEntity()

	component := graphicsComponent{
		posX:       0,
		posY:       0,
		entityKind: entityKindColorPalette,
	}
	ui.graphics.addComponent(e, component)

	return e
}

func makeButton(ui uiSystem, txt string, colorDisabled, colorIdle, colorPressed, colorHovered colorID) entity {
	e := newEntity()

	stateComponent := stateComponent{kind: entityKindButton, state: 0}
	ui.states.addComponent(e, stateComponent)

	x := 100
	y := 100

	interactableComponent := interactableComponent{
		tag:        e,
		posX:       x,
		posY:       y,
		width:      100,
		height:     50,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	ui.interactables.addComponent(e, interactableComponent)

	colors := [8]colorID{colorDisabled, colorIdle, colorPressed, colorHovered}
	graphicsComponent := graphicsComponent{
		posX:       x,
		posY:       y,
		text:       txt,
		textColor:  colorWhite,
		colors:     colors,
		width:      100,
		height:     50,
		bgColor:    colors[btnStateIdle],
		entityKind: entityKindButton,
	}
	ui.graphics.addComponent(e, graphicsComponent)

	return e
}
