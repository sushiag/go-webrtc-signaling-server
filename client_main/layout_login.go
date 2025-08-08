package main

import (
	"gioui.org/layout"
)

func layoutLoginPage(
	gtx layout.Context,
	bboxes system[boundingBoxComponent],
	loginPage loginPageMetadata,
) {
	windowBB := bbFromGtx(gtx)
	title, _ := bboxes.getComponentRef(loginPage.loginTitle)
	username, _ := bboxes.getComponentRef(loginPage.usernameInput)
	password, _ := bboxes.getComponentRef(loginPage.passwordInput)
	loginBtn, _ := bboxes.getComponentRef(loginPage.loginBtn)
	signupBtn, _ := bboxes.getComponentRef(loginPage.signupBtn)

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
