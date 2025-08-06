package main

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"
)

type graphicsComponent struct {
	posX       int
	posY       int
	text       string
	colors     [8]uint8
	entityID   uint32
	width      int
	height     int
	bgColor    uint8
	textColor  uint8
	textColors [2]uint8
	entityKind entityKind
	isDisabled bool
}

func (g graphicsComponent) draw(gtx layout.Context, textShaper *text.Shaper) {
	switch g.entityKind {
	case entityKindColorPalette:
		drawColorPalette(g, gtx)
	case entityKindButton:
		drawButton(g, gtx, textShaper)
	}
}

func drawColorPalette(g graphicsComponent, gtx layout.Context) {
	const itemDisplaySize = 50
	const width = int(_nColors) * itemDisplaySize
	const height = itemDisplaySize

	x0 := g.posX
	y0 := g.posY
	x1 := g.posX + width
	y1 := g.posY + height
	defer clip.Rect(image.Rect(int(x0), int(y0), int(x1), int(y1))).Push(gtx.Ops).Pop()

	for i := range len(colorPalette) {
		xOffset := int(x0) + (i * itemDisplaySize)

		bounds := clip.Rect(image.Rect(xOffset, 0, itemDisplaySize+xOffset, itemDisplaySize)).Push(gtx.Ops)
		paint.Fill(gtx.Ops, colorPalette[i])
		bounds.Pop()
	}

}

func drawButton(g graphicsComponent, gtx layout.Context, textShaper *text.Shaper) {
	x0 := g.posX
	y0 := g.posY
	x1 := g.posX + g.width
	y1 := g.posY + g.height
	defer clip.Rect(image.Rect(int(x0), int(y0), int(x1), int(y1))).Push(gtx.Ops).Pop()

	paint.Fill(gtx.Ops, colorPalette[g.bgColor])

	if g.text != "" {
		const defaultTextSize = 16
		textSize := fixed.I(gtx.Sp(defaultTextSize))

		textParams := text.Parameters{
			Alignment: text.Middle,
			PxPerEm:   textSize,
			MaxLines:  1,
			Truncator: "...",
			MinWidth:  int(g.width),
			MaxWidth:  int(g.width),
		}

		drawCalls, _, height := renderText(gtx, textShaper, textParams, g.text, colorPalette[g.textColor], unit.Sp(textSize))

		yOffset := (float32(g.height) - height) / 2.5
		offset := image.Point{X: x0, Y: y0 + int(yOffset)}
		offsetStack := op.Offset(offset).Push(gtx.Ops)

		for _, drawCall := range drawCalls {
			drawCall.Add(gtx.Ops)
		}

		offsetStack.Pop()
	}
}
