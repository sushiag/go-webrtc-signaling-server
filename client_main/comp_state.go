package main

import "gioui.org/io/pointer"

type stateComponent struct {
	entityID uint32
	kind     entityKind
	state    uint8
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

func (s *stateComponent) processBtnEvent(event pointer.Kind) buttonState {
	switch event {
	case pointer.Press:
		s.state = btnStatePressed
	case pointer.Enter:
		s.state = btnStateHovered
	case pointer.Leave:
		s.state = btnStateIdle
	case pointer.Release:
		s.state = btnStateHovered
	}
	return s.state
}
