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
		labelConfig{
			width:     300,
			height:    50,
			text:      "better than discord 8-D",
			textColor: colorWhite,
			bgColor:   colorTransparent,
		},
	)
	appState.login.usernameInput = makeTextInput(
		systems,
		textInputConfig{
			width:             300,
			height:            50,
			placeholderText:   "username",
			textColorDisabled: colorGray,
			textColorIdle:     colorWhite,
			textColorHovered:  colorBlack,
			textColorFocused:  colorWhite,
			colorDisabled:     colorPurpleDark,
			colorIdle:         colorPurpleDarker,
			colorHovered:      colorWhite,
			colorFocused:      colorPurpleLight,
		},
	)
	appState.login.passwordInput = makeTextInput(
		systems,
		textInputConfig{
			width:             300,
			height:            50,
			placeholderText:   "password",
			textColorDisabled: colorGray,
			textColorIdle:     colorWhite,
			textColorHovered:  colorBlack,
			textColorFocused:  colorWhite,
			colorDisabled:     colorPurpleDark,
			colorIdle:         colorPurpleDarker,
			colorHovered:      colorWhite,
			colorFocused:      colorPurpleLight,
		},
	)
	appState.login.loginBtn = makeButton(
		systems,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "login",
			colorDisabled:     colorPurpleDarker,
			colorIdle:         colorPink,
			colorPressed:      colorPurpleDark,
			colorHovered:      colorWhite,
			textColorDisabled: colorGray,
			textColorIdle:     colorWhite,
			textColorPressed:  colorWhite,
			textColorHovered:  colorBlack,
		},
	)
	appState.login.signupBtn = makeButton(
		systems,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "sign up",
			colorDisabled:     colorPurpleDarker,
			colorIdle:         colorPurpleLight,
			colorPressed:      colorPurpleLight,
			colorHovered:      colorWhite,
			textColorDisabled: colorGray,
			textColorIdle:     colorWhite,
			textColorPressed:  colorWhite,
			textColorHovered:  colorBlack,
		},
	)
}
