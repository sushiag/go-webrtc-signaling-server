package main

import (
	"log"

	"gioui.org/io/pointer"
)

type state = uint8

type stateComponent struct {
	kind  bundleKind
	state uint8
}

type buttonState = uint8

const (
	btnStateDisabled buttonState = iota
	btnStateIdle
	btnStatePressed
	btnStateHovered
)

type labelState = uint8

const (
	lblStateDisabled labelState = iota
	lblStatetIdle
	lblStatePressed
	lblStateHovered
	lblStateFocused
)

type textInputState = uint8

const (
	txtInputDisabled textInputState = iota
	txtInputStateIdle
	txtInputStateHovered
	txtInputStateFocused
)

func (g stateComponent) ptrInteraction(eventKind pointer.Kind) state {
	switch g.kind {
	case bundleButton:
		log.Println("btn input")
		return g.processBtnPtrEvent(eventKind)
	case bundleTextInput:
		return g.processTextInputPtrEvent(eventKind)
	default:
		log.Panicln("[ERR] ptrInteraction function not defined for bundle:", g.kind)
		return 0
	}
}

func (s stateComponent) processBtnPtrEvent(event pointer.Kind) buttonState {
	switch event {
	case pointer.Press:
		return btnStatePressed
	case pointer.Enter:
		return btnStateHovered
	case pointer.Leave:
		return btnStateIdle
	case pointer.Release:
		// TODO: we need to check if the pointer is still inside the
		// box when released
		return btnStateHovered
	default:
		return s.state
	}
}

func (s stateComponent) processTextInputPtrEvent(event pointer.Kind) textInputState {
	switch event {
	case pointer.Press:
		return txtInputStateFocused
	case pointer.Enter:
		if s.state == txtInputStateFocused {
			return s.state
		} else {
			return txtInputStateHovered
		}
	case pointer.Leave:
		if s.state == txtInputStateFocused {
			return s.state
		} else {
			return txtInputStateIdle
		}
	default:
		return s.state
	}
}
