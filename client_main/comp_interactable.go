package main

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

type interactableComponent struct {
	ptrEvFlags pointer.Kind
	tag        entity
	focusable  bool
	isDisabled bool
}

func (c interactableComponent) declareEventRegion(gtx layout.Context, bbox boundingBoxComponent) {
	defer bbox.clip().Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, c.tag)
}

func (c interactableComponent) getEventFilters(filters *[]event.Filter) {
	if c.ptrEvFlags != 0 {
		pointerFilter := pointer.Filter{
			Target:  c.tag,
			Kinds:   c.ptrEvFlags,
			ScrollX: pointer.ScrollRange{Min: -100, Max: 100},
			ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
		}
		*filters = append(*filters, pointerFilter)
	}

	if c.focusable {
		*filters = append(*filters,
			key.FocusFilter{Target: c.tag},
			key.Filter{
				Focus:    c.tag,
				Required: 0,
				Optional: 0,
				Name:     "",
			},
		)
	}
}
