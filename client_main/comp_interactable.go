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
	keyEv      bool
	isDisabled bool
}

func (c interactableComponent) declareEventRegion(gtx layout.Context, bbox boundingBoxComponent) {
	defer bbox.clip().Push(gtx.Ops).Pop()

	key.InputHintOp{
		Tag:  c.tag,
		Hint: key.HintAny,
	}.Add(gtx.Ops)

	filters := make([]event.Filter, 0, 2)

	if c.ptrEvFlags != 0 {
		pointerFilter := pointer.Filter{
			Target:  c.tag,
			Kinds:   c.ptrEvFlags,
			ScrollX: pointer.ScrollRange{Min: -100, Max: 100},
			ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
		}
		filters = append(filters, pointerFilter)
	}

	if c.keyEv {
		keyFilter := key.Filter{
			Focus:    nil,
			Required: 0,
			Optional: 0,
			Name:     "",
		}
		filters = append(filters, keyFilter)
	}

	event.Op(gtx.Ops, c.tag)
}

func (c interactableComponent) getEventFilters(filters []event.Filter) []event.Filter {
	if c.ptrEvFlags != 0 {
		pointerFilter := pointer.Filter{
			Target:  c.tag,
			Kinds:   c.ptrEvFlags,
			ScrollX: pointer.ScrollRange{Min: -100, Max: 100},
			ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
		}
		filters = append(filters, pointerFilter)
	}

	if c.keyEv {
		keyFilter := key.Filter{
			Focus:    nil,
			Required: 0,
			Optional: 0,
			Name:     "",
		}
		filters = append(filters, keyFilter)
	}

	return filters
}
