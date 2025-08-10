package main

func initLoginPageEntities(appState *appState, sys system) {
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

func initMainPageEntities(appState *appState, sys system) {
	appState.main.logoDisp = makeLabel(
		sys,
		labelConfig{
			width:     300,
			height:    50,
			text:      "8-D",
			textColor: colorWhite,
			bgColor:   colorTransparent,
		},
	)
	appState.main.usernameDisp = makeLabel(
		sys,
		labelConfig{
			width:     300,
			height:    50,
			text:      "Chester#1",
			textColor: colorWhite,
			bgColor:   colorTransparent,
		},
	)
	appState.main.serversButton = makeButton(
		sys,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "Servers",
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
	appState.main.messageButton = makeButton(
		sys,
		buttonConfig{
			width:             100,
			height:            50,
			text:              "Messages",
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
