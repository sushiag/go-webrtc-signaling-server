package main

import (
	"gioui.org/layout"
)

func layoutMainPage(
	gtx layout.Context,
	sys system,
	mainPage mainPageState,
) {
	windowBB := bbFromGtx(gtx)
	logo := sys.getBBoxComponentRef(mainPage.logoDisp)
	username := sys.getBBoxComponentRef(mainPage.usernameDisp)
	serversBtn := sys.getBBoxComponentRef(mainPage.serversButton)
	msgsBtn := sys.getBBoxComponentRef(mainPage.messageButton)

	// main
	topBar := newBBox(windowBB.pos[0], windowBB.pos[1], windowBB.size[0], 80)
	layoutFlex(
		windowBB,
		flexVertical,
		flexSpaceEnd,
		flexAlignMiddle,
		[]flexItem{
			{&topBar, 0.0, margin{0, 0, 0, 0}},
			{serversBtn, 0.0, margin{0, 0, 0, 0}},
			{msgsBtn, 0.0, margin{0, 0, 0, 0}},
		},
	)

	// top bar
	layoutFlex(
		topBar,
		flexHorizontal,
		flexSpaceBetween,
		flexAlignStart,
		[]flexItem{
			{logo, 0.0, margin{0, 0, 0, 0}},
			{username, 0.0, margin{0, 0, 0}},
		},
	)
}
