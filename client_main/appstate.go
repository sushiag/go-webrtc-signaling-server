package main

type appState struct {
	colorPalette entity
	currentPage  appPage
	login        loginPageMetadata
	main         mainPageMetadata
}

type appPage = uint8

const (
	apploginPage appPage = iota
	appMainPage
)

type loginPageMetadata struct {
	loginBtn entity
}

type mainPageMetadata struct {
}
