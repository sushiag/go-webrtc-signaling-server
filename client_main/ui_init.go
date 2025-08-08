package main

import (
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
	appState.login.loginTitle = makeLabel(
		systems,
		300, 50,
		"better than discord 8-D",
		colorWhite,
		colorTransparent,
	)
	appState.login.loginBtn = makeButton(
		systems,
		"login",
		100, 50,
		colorPurpleDarker, colorPurpleLight, colorPurpleDark, colorWhite,
		colorBlack, colorWhite, colorWhite, colorBlack,
	)
	appState.login.signupBtn = makeButton(
		systems,
		"sign up",
		100, 50,
		colorPurpleDarker, colorPurpleDark, colorPurpleLight, colorWhite,
		colorBlack, colorWhite, colorWhite, colorBlack,
	)
}
