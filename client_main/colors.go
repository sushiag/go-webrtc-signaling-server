package main

import "image/color"

type colorID = uint8

const (
	colorMissing colorID = iota
	colorBlack
	colorWhite
	colorGray
	colorPurpleLight
	colorPurpleDark
	colorPurpleDarker
	colorPink
	colorTransparent
	_nColors
)

var colorPalette = []color.NRGBA{
	{R: 0xe5, G: 0xff, B: 0x06, A: 0xff}, // yellow (used to indicate unset colors)
	{R: 0x0, G: 0x0, B: 0x0, A: 0xff},    // black
	{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // white
	{R: 0x76, G: 0x5c, B: 0x69, A: 0xff}, // gray
	{R: 0x4d, G: 0x36, B: 0x53, A: 0xff}, // purple light
	{R: 0x36, G: 0x24, B: 0x3b, A: 0xff}, // purple dark
	{R: 0x27, G: 0x1c, B: 0x2a, A: 0xff}, // purple darker
	{R: 0x97, G: 0x46, B: 0x6a, A: 0xff}, // pink
	{R: 0x00, G: 0x00, B: 0x00, A: 0x00}, // transparent
}
