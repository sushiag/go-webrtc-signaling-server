package main

import (
	"gioui.org/io/pointer"
)

type state = uint8

type stateComponent struct {
	bundleKind bundleKind
	state      state
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

func getNextBtnState(currentState state, event pointer.Event) state {
	if currentState == btnStateDisabled {
		return currentState
	}

	switch event.Kind {
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
	}

	return currentState
}

func getNextTxtInputState(currentState state, event pointer.Event) state {
	if currentState == txtInputDisabled {
		return currentState
	}

	switch event.Kind {
	case pointer.Press:
		return txtInputStateFocused
	case pointer.Enter:
		if currentState != txtInputStateFocused {
			return txtInputStateHovered
		}
	case pointer.Leave:
		if currentState != txtInputStateFocused {
			return txtInputStateIdle
		}
	}

	return currentState
}
