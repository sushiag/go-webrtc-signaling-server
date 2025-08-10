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

func (s *stateComponent) handlePtrInteraction(eventKind pointer.Kind) {
	switch s.kind {
	case bundleButton:
		s.handleBtnPtrEvent(eventKind)
	case bundleTextInput:
		s.handleTextInputPtrEvent(eventKind)
	default:
		log.Panicln("[ERR] ptrInteraction function not defined for bundle:", s.kind)
	}
}

func (s *stateComponent) handleBtnPtrEvent(event pointer.Kind) {
	switch event {
	case pointer.Press:
		s.state = btnStatePressed
	case pointer.Enter:
		s.state = btnStateHovered
	case pointer.Leave:
		s.state = btnStateIdle
	case pointer.Release:
		// TODO: we need to check if the pointer is still inside the
		// box when released
		s.state = btnStateHovered
	}
}

func (s *stateComponent) handleTextInputPtrEvent(event pointer.Kind) {
	switch event {
	case pointer.Press:
		s.state = txtInputStateFocused
	case pointer.Enter:
		if s.state == txtInputStateFocused {
			s.state = s.state
		} else {
			s.state = txtInputStateHovered
		}
	case pointer.Leave:
		if s.state == txtInputStateFocused {
			s.state = s.state
		} else {
			s.state = txtInputStateIdle
		}
	}
}
