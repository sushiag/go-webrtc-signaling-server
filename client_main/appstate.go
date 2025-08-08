package main

type appState struct {
	login           loginPageMetadata
	colorPalette    entity
	focusedInput    entity
	currentPage     appPage
	hasFocusedInput bool
	main            mainPageMetadata
}

type appPage = uint8

const (
	apploginPage appPage = iota
	appMainPage
)

type loginPageMetadata struct {
	loginTitle    entity
	usernameInput entity
	passwordInput entity
	loginBtn      entity
	signupBtn     entity
}

type mainPageMetadata struct {
}
