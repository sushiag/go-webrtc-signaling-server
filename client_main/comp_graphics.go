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

// TODO: rename this
type graphicsKind = uint8

const (
	gkColorPalette graphicsKind = iota
	entityKindLabel
	gkButton
)

type graphicsComponent struct {
	text       string
	colors     [8]uint8
	bgColor    uint8
	textColor  uint8
	textColors [2]uint8
	kind       graphicsKind
	isDisabled bool
}

func (g graphicsComponent) draw(gtx layout.Context, bb boundingBoxComponent, textShaper *text.Shaper) {
	switch g.kind {
	case gkColorPalette:
		drawColorPalette(gtx, bb)
	case gkButton:
		drawButton(gtx, bb, g, textShaper)
	}
}

func drawColorPalette(gtx layout.Context, bb boundingBoxComponent) {
	const itemDisplaySize = 50
	const width = int(_nColors) * itemDisplaySize
	const height = itemDisplaySize

	bb.size[0] = width
	bb.size[1] = height
	defer bb.clip().Push(gtx.Ops).Pop()

	for i := range len(colorPalette) {
		xOffset := int(bb.pos[0]) + (i * itemDisplaySize)

		bounds := clip.Rect(image.Rect(xOffset, 0, itemDisplaySize+xOffset, itemDisplaySize)).Push(gtx.Ops)
		paint.Fill(gtx.Ops, colorPalette[i])
		bounds.Pop()
	}

}

func drawButton(
	gtx layout.Context,
	bb boundingBoxComponent,
	g graphicsComponent,
	textShaper *text.Shaper,
) {
	defer bb.clip().Push(gtx.Ops).Pop()

	paint.Fill(gtx.Ops, colorPalette[g.bgColor])

	if g.text != "" {
		const defaultTextSize = 16
		textSize := fixed.I(gtx.Sp(defaultTextSize))

		textParams := text.Parameters{
			Alignment: text.Middle,
			PxPerEm:   textSize,
			MaxLines:  1,
			Truncator: "...",
			MinWidth:  int(bb.size[0]),
			MaxWidth:  int(bb.size[0]),
		}

		drawCalls, _, height := renderText(gtx, textShaper, textParams, g.text, colorPalette[g.textColor], unit.Sp(textSize))

		yOffset := (float32(bb.size[1]) - height) / 2.5
		offset := image.Point{X: bb.pos[0], Y: bb.pos[1] + int(yOffset)}
		offsetStack := op.Offset(offset).Push(gtx.Ops)

		for _, drawCall := range drawCalls {
			drawCall.Add(gtx.Ops)
		}

		offsetStack.Pop()
	}
}
