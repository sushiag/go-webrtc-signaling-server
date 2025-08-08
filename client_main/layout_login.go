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
	title, _, _ := bboxes.getComponentRef(loginPage.loginTitle)
	loginBtn, _, _ := bboxes.getComponentRef(loginPage.loginBtn)
	signupBtn, _, _ := bboxes.getComponentRef(loginPage.signupBtn)

	flexItems := []flexItem{
		{title, 0.0, margin{0, 0, 0, 100}},
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
