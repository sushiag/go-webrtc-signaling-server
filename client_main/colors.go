package main

import "image/color"

type colorID = uint8

const (
	colorWhite colorID = iota
	colorBlack
	colorGray
	colorPurpleLight
	colorPurpleDark
	colorPurpleDarker
	colorPink
	_nColors
)

var colorPalette = []color.NRGBA{
	{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, // white
	{R: 0x0, G: 0x0, B: 0x0, A: 0xFF},    // black
	{R: 0x76, G: 0x5c, B: 0x69, A: 0xFF}, // gray
	{R: 0x4d, G: 0x36, B: 0x53, A: 0xFF}, // purple light
	{R: 0x36, G: 0x24, B: 0x3b, A: 0xFF}, // purple dark
	{R: 0x27, G: 0x1c, B: 0x2a, A: 0xFF}, // purple darker
	{R: 0x97, G: 0x46, B: 0x6a, A: 0xFF}, // pink
}
