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
	appState.login.usernameInput = makeTextInput(
		systems,
		300, 50,
		"username",
		colorGray, colorWhite, colorBlack, colorWhite,
		colorPurpleDark, colorPurpleDarker, colorWhite, colorPurpleLight,
	)
	appState.login.passwordInput = makeTextInput(
		systems,
		300, 50,
		"password",
		colorGray, colorWhite, colorBlack, colorWhite,
		colorPurpleDark, colorPurpleDarker, colorWhite, colorPurpleLight,
	)
	appState.login.loginBtn = makeButton(
		systems,
		"login",
		100, 50,
		colorPurpleDarker, colorPink, colorPurpleDark, colorWhite,
		colorGray, colorWhite, colorWhite, colorBlack,
	)
	appState.login.signupBtn = makeButton(
		systems,
		"sign up",
		100, 50,
		colorPurpleDarker, colorPurpleLight, colorPurpleLight, colorWhite,
		colorGray, colorWhite, colorWhite, colorBlack,
	)
}
