package main

import "gioui.org/io/pointer"

type stateComponent struct {
	kind  entityKind
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

// Returns the next state after given some event
func (s stateComponent) processBtnEvent(event pointer.Kind) buttonState {
	switch event {
	case pointer.Press:
		return btnStatePressed
	case pointer.Enter:
		return btnStateHovered
	case pointer.Leave:
		return btnStateIdle
	case pointer.Release:
		return btnStateHovered
	default:
		return s.state
	}
}
