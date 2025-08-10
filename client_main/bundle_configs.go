package main

type bundleKind = uint8

const (
	bundleLabel bundleKind = iota
	bundleButton
	bundleTextInput
	bundleColorPalette
)

type labelConfig struct {
	width     int
	height    int
	text      string
	textColor colorID
	bgColor   colorID
}

type buttonConfig struct {
	width             int
	height            int
	text              string
	initState         buttonState
	colorDisabled     colorID
	colorIdle         colorID
	colorPressed      colorID
	colorHovered      colorID
	textColorDisabled colorID
	textColorIdle     colorID
	textColorPressed  colorID
	textColorHovered  colorID
}

type textInputConfig struct {
	width             int
	height            int
	placeholderText   string
	initState         labelState
	textColorDisabled colorID
	textColorIdle     colorID
	textColorHovered  colorID
	textColorFocused  colorID
	colorDisabled     colorID
	colorIdle         colorID
	colorHovered      colorID
	colorFocused      colorID
}
