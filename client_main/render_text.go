package main

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"
)

// Utility function for rendering text
//
// Returns (glyphDrawCalls, totalWidth, totalHeight)
func renderText(
	gtx layout.Context,
	textShaper *text.Shaper,
	textParams text.Parameters,
	txt string,
	textColor color.NRGBA,
	textSize unit.Sp,
) ([]op.CallOp, float32, float32) {
	textShaper.LayoutString(
		textParams,
		txt,
	)

	paint.ColorOp{Color: textColor}.Add(gtx.Ops)

	textWidth := fixed.Int26_6(0)
	textHeight := fixed.Int26_6(0)

	var callOps [32]op.CallOp
	var glyphs [32]text.Glyph

	drawCalls := callOps[:0]
	line := glyphs[:0]
	maxGlyphHeight := fixed.Int26_6(0)
	for {
		glyph, ok := textShaper.NextGlyph()
		if !ok {
			break
		}

		// calculate position start
		line = append(line, glyph)
		xPos := float32(glyph.X) / 64.0
		yPos := float32(glyph.Y)
		offset := f32.Point{X: xPos, Y: yPos}

		textWidth += glyph.Advance
		glyphHeight := glyph.Ascent - glyph.Descent
		if glyphHeight > maxGlyphHeight {
			maxGlyphHeight = glyphHeight
		}
		if glyph.Flags&text.FlagLineBreak == text.FlagLineBreak {
			textHeight += maxGlyphHeight
			maxGlyphHeight = 0
		}
		// calculate position end

		// drawing start
		macro := op.Record(gtx.Ops)

		transform := op.Affine(f32.Affine2D{}.Offset(offset)).Push(gtx.Ops)
		path := textShaper.Shape(line)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		if call := textShaper.Bitmaps(line); call != (op.CallOp{}) {
			call.Add(gtx.Ops)
		}
		transform.Pop()

		c := macro.Stop()
		drawCalls = append(drawCalls, c)
		// drawing end

		line = line[:0]
	}

	labelWidthf32 := float32(textWidth) / 64.0
	labelHeightf32 := float32(textHeight) / 64.0
	if labelHeightf32 == 0.0 {
		labelHeightf32 = float32(textSize) / 64.0
	}

	return drawCalls, labelWidthf32, labelHeightf32
}
