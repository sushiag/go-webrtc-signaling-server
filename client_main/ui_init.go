package main

import (
	"gioui.org/io/pointer"
	"gioui.org/text"
)

type systems struct {
	states        *system[stateComponent]
	bBoxes        *system[boundingBoxComponent]
	interactables *system[interactableComponent]
	graphics      *system[graphicsComponent]
	textShaper    *text.Shaper
}

func initEntities(appState *appState, systems systems) {
	appState.colorPalette = makeColorPalette(systems)
	appState.login.loginBtn = makeButton(
		systems,
		"login",
		100, 50,
		colorPurpleDarker, colorPurpleLight, colorPurpleDark, colorWhite,
	)
	appState.login.signupBtn = makeButton(
		systems,
		"sign up",
		100, 50,
		colorPurpleDarker, colorPurpleDark, colorPurpleLight, colorWhite,
	)
	appState.login.anotherBtn = makeButton(
		systems,
		"big",
		200, 100,
		colorPurpleDarker, colorPurpleLight, colorPurpleDark, colorWhite,
	)
}

func makeColorPalette(ui systems) entity {
	e := newEntity()

	graphics := graphicsComponent{
		kind:       gkColorPalette,
		isDisabled: true,
	}
	ui.graphics.addComponent(e, graphics)

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{0, 0}}
	ui.bBoxes.addComponent(e, boundingBox)

	return e
}

func makeButton(
	ui systems,
	txt string,
	width, height int,
	colorDisabled, colorIdle, colorPressed, colorHovered colorID,
) entity {
	e := newEntity()

	state := stateComponent{kind: gkButton, state: 0}
	ui.states.addComponent(e, state)

	boundingBox := boundingBoxComponent{size: [2]int{width, height}}
	ui.bBoxes.addComponent(e, boundingBox)

	interactable := interactableComponent{
		tag:        e,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	ui.interactables.addComponent(e, interactable)

	colors := [8]colorID{colorDisabled, colorIdle, colorPressed, colorHovered}
	graphics := graphicsComponent{
		text:      txt,
		textColor: colorWhite,
		colors:    colors,
		bgColor:   colors[btnStateIdle],
		kind:      gkButton,
	}
	ui.graphics.addComponent(e, graphics)

	return e
}
