package main

import (
	"math"
)

func layoutCenter(outer boundingBoxComponent, inner *boundingBoxComponent) {
	inner.pos[0] = outer.pos[0] + ((outer.size[0] - inner.size[0]) / 2)
	inner.pos[1] = outer.pos[1] + ((outer.size[1] - inner.size[1]) / 2)
}

type flexItem struct {
	bb   *boundingBoxComponent
	flex float64
}

type flexDirection = uint8
type flexSpacing = uint8
type flexAlignment = uint8

const (
	flexHorizontal flexDirection = iota
	flexVertical
)

const (
	flexAlignMiddle flexAlignment = iota
	flexAlignStart
	flexAlignEnd
)

const (
	flexSpaceBetween flexSpacing = iota
	flexSpaceAround
	flexSpaceStart
	flexSpaceEnd
	flexSpaceSide
)

// NOTE: will change the sizes of the bounding boxes
func layoutFlex(
	parent boundingBoxComponent,
	direction flexDirection,
	spacing flexSpacing,
	alignment flexAlignment,
	items []flexItem,
) {
	primaryAxis := int(direction)
	secondaryAxis := 1 - primaryAxis

	totalFlex := 0.0
	availableSpace := float64(parent.size[primaryAxis])

	for _, c := range items {
		if c.flex > 0 {
			totalFlex += c.flex
		} else {
			availableSpace -= float64(c.bb.size[primaryAxis])
		}
	}

	secondaryAxisMiddle := parent.size[secondaryAxis] / 2
	spacingIncrements := availableSpace / totalFlex

	offset := 0.0
	spaceAmount := 0.0

	if totalFlex == 0.0 {
		switch spacing {
		case flexSpaceStart:
			offset = math.Round(availableSpace)
		case flexSpaceBetween:
			spaceAmount = math.Round(availableSpace / float64(len(items)-1))
		case flexSpaceAround:
			spaceAmount = math.Round(availableSpace / float64(len(items)+1))
			offset = spaceAmount
		case flexSpaceSide:
			spaceAmount = math.Round(availableSpace / 2)
			offset = spaceAmount
		}
	}

	for _, item := range items {
		spaceOccupied := float64(item.bb.size[primaryAxis])
		if item.flex > 0.0 {
			spaceOccupied = math.Round(spacingIncrements * item.flex)
		}

		item.bb.size[primaryAxis] = int(spaceOccupied)
		item.bb.pos[primaryAxis] = int(offset)

		switch alignment {
		case flexAlignMiddle:
			item.bb.pos[secondaryAxis] = secondaryAxisMiddle - (item.bb.size[secondaryAxis] / 2)
		case flexAlignStart:
			item.bb.pos[secondaryAxis] = parent.pos[secondaryAxis]
		case flexAlignEnd:
			item.bb.pos[secondaryAxis] = parent.size[secondaryAxis] - item.bb.size[secondaryAxis]
		}

		switch spacing {
		case flexSpaceEnd, flexSpaceStart, flexSpaceSide:
			offset += spaceOccupied
		case flexSpaceBetween, flexSpaceAround:
			offset += spaceOccupied + spaceAmount
		}
	}
}
