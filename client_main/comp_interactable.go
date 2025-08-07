package main

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

type interactableComponent struct {
	tag        entity
	posX       int
	posY       int
	width      int
	height     int
	ptrEvFlags pointer.Kind
	keyEv      bool
	isDisabled bool
}

type interactionEvent struct {
	entityID uint32
	kind     event.Event
}

func (c interactableComponent) captureEvents(gtx layout.Context, outEvents *[]interactionEvent) {
	x0 := c.posX
	y0 := c.posY
	x1 := c.posX + c.width
	y1 := c.posY + c.height
	defer clip.Rect(image.Rect(int(x0), int(y0), int(x1), int(y1))).Push(gtx.Ops).Pop()

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

	// TODO: move this out
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		*outEvents = append(*outEvents, interactionEvent{c.tag, ev})
	}
}
