package main

import (
	"image"
	"log"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"
)

// left, right, top, bottom
type border = [8]int

type graphicsComponent struct {
	text            string
	placeholderText string
	textColors      [8]colorID
	bgColors        [8]colorID
	bgColor         uint8
	border          border
	borderColor     colorID
	textColor       uint8
	kind            bundleKind
	isDisabled      bool
}

func (g graphicsComponent) draw(gtx layout.Context, bb boundingBoxComponent, textShaper *text.Shaper) {
	switch g.kind {
	case bundleLabel, bundleButton:
		drawSquareWithText(gtx, bb, g, textShaper)
	case bundleTextInput:
		drawTextInput(gtx, bb, g, textShaper)
	case bundleColorPalette:
		drawColorPalette(gtx, bb)
	default:
		log.Panicln("[ERR] draw function not defined for bundle:", g.kind)
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

func drawSquareWithText(
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
			MaxLines:  0,
			Truncator: "...",
			MinWidth:  int(bb.size[0]),
			MaxWidth:  int(bb.size[0]),
		}

		drawCalls, _, height := renderText(gtx, textShaper, textParams, g.text, colorPalette[g.textColor], unit.Sp(textSize))

		yOffset := (float64(bb.size[1]) - height) / 2.5
		offset := image.Point{X: bb.pos[0], Y: bb.pos[1] + int(yOffset)}
		offsetStack := op.Offset(offset).Push(gtx.Ops)

		for _, drawCall := range drawCalls {
			drawCall.Add(gtx.Ops)
		}

		offsetStack.Pop()
	}
}

func drawTextInput(
	gtx layout.Context,
	bb boundingBoxComponent,
	g graphicsComponent,
	textShaper *text.Shaper,
) {
	defer bb.clip().Push(gtx.Ops).Pop()
	// fill border
	paint.Fill(gtx.Ops, colorPalette[g.borderColor])

	// fill non-border
	borderBox := bb
	borderBox.pos[0] += g.border[0]
	borderBox.pos[1] += g.border[1]
	borderBox.size[0] -= g.border[0] + g.border[2]
	borderBox.size[1] -= g.border[1] + g.border[3]
	borderClipStack := borderBox.clip().Push(gtx.Ops)
	paint.Fill(gtx.Ops, colorPalette[g.bgColor])
	borderClipStack.Pop()

	const defaultTextSize = 16
	textSize := fixed.I(gtx.Sp(defaultTextSize))

	textParams := text.Parameters{
		Alignment: text.Middle,
		PxPerEm:   textSize,
		MaxLines:  0,
		Truncator: "...",
		MinWidth:  int(bb.size[0]),
		MaxWidth:  int(bb.size[0]),
	}

	drawCalls, _, height := renderText(gtx, textShaper, textParams, g.text, colorPalette[g.textColor], unit.Sp(textSize))

	yOffset := (float64(bb.size[1]) - height) / 2.5
	offset := image.Point{X: bb.pos[0], Y: bb.pos[1] + int(yOffset)}
	offsetStack := op.Offset(offset).Push(gtx.Ops)

	for _, drawCall := range drawCalls {
		drawCall.Add(gtx.Ops)
	}

	offsetStack.Pop()
}
