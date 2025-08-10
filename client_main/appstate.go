package main

type appState struct {
	login        loginPageState
	focus        focusState
	colorPalette entity
	currentPage  appPage
	main         mainPageState
}

type appPage = uint8

const (
	apploginPage appPage = iota
	appMainPage
)

type focusState struct {
	focusedEntity   entity
	hasFocusedInput bool
}

type loginPageState struct {
	textInputText     string
	passwordInputText string
	loginTitle        entity
	usernameInput     entity
	passwordInput     entity
	loginBtn          entity
	signupBtn         entity
}

type mainPageState struct {
}
