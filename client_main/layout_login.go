package main

import "gioui.org/layout"

func layoutLoginPage(
	gtx layout.Context,
	bboxes system[boundingBoxComponent],
	loginPage loginPageMetadata,
) {
	windowBB := bbFromGtx(gtx)
	loginBtnBB, loginBtnIdx, _ := bboxes.getComponent(loginPage.loginBtn)
	signupBtnBB, signupBtnIdx, _ := bboxes.getComponent(loginPage.signupBtn)
	anotherBtnBB, anotherBtnIdx, _ := bboxes.getComponent(loginPage.anotherBtn)
	layoutFlex(
		windowBB,
		flexVertical,
		flexSpaceEnd,
		flexAlignMiddle,
		[]flexItem{{&loginBtnBB, 0.0}, {&signupBtnBB, 0.0}, {&anotherBtnBB, 0.0}},
	)
	bboxes.updateComponent(loginBtnIdx, loginBtnBB)
	bboxes.updateComponent(signupBtnIdx, signupBtnBB)
	bboxes.updateComponent(anotherBtnIdx, anotherBtnBB)
}
