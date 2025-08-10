package main

func initLoginPageEntities(appState *appState, sys system) {
	appState.colorPalette = makeColorPalette(sys)
	appState.login.loginTitle = makeLabel(
		sys,
		labelConfig{
			width:     300,
			height:    50,
			text:      "better than discord 8-D",
			textColor: colorWhite,
			bgColor:   colorTransparent,
		},
	)
	appState.login.usernameInput = makeTextInput(
		sys,
		textInputConfig{
			width:             300,
			height:            50,
			initState:         txtInputStateIdle,
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
		sys,
		textInputConfig{
			width:             300,
			height:            50,
			placeholderText:   "password",
			initState:         txtInputStateIdle,
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
		sys,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "login",
			initState:         btnStateIdle,
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
		sys,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "sign up",
			initState:         btnStateIdle,
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
