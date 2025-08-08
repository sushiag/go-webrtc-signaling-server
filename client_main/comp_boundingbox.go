package main

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
)

type boundingBoxComponent struct {
	// X, Y
	pos [2]int
	// Width, Height
	size [2]int
}

// Helper function for creating a clip.Rect in the given region
func (bb *boundingBoxComponent) clip() clip.Rect {
	x0 := bb.pos[0]
	y0 := bb.pos[1]
	x1 := bb.pos[0] + bb.size[0]
	y1 := bb.pos[1] + bb.size[1]
	return clip.Rect(image.Rect(int(x0), int(y0), int(x1), int(y1)))
}

// Creates a bounding box from a layout context
func bbFromGtx(gtx layout.Context) boundingBoxComponent {
	return boundingBoxComponent{
		pos:  [2]int{0, 0},
		size: [2]int{gtx.Constraints.Max.X, gtx.Constraints.Max.Y},
	}
}
