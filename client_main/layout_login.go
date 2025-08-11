package main

import (
	"gioui.org/layout"
)

func layoutLoginPage(
	gtx layout.Context,
	app *appState,
	sys system,
) {
	loginPage := &app.login

	windowBB := bbFromGtx(gtx)
	title := sys.getBBoxComponentRef(loginPage.loginTitle)
	username := sys.getBBoxComponentRef(loginPage.usernameInput)
	password := sys.getBBoxComponentRef(loginPage.passwordInput)
	loginBtn := sys.getBBoxComponentRef(loginPage.loginBtn)
	signupBtn := sys.getBBoxComponentRef(loginPage.signupBtn)

	flexItems := []flexItem{
		{title, 0.0, margin{0, 0, 0, 60}},
		{username, 0.0, margin{0, 0, 0, 30}},
		{password, 0.0, margin{0, 0, 0, 50}},
		{loginBtn, 0.0, margin{0, 0, 0, 30}},
		{signupBtn, 0.0, margin{0, 0, 0, 0}},
	}

	layoutFlex(
		windowBB,
		flexVertical,
		flexSpaceSide,
		flexAlignMiddle,
		flexItems,
	)
}
