package main

import "gioui.org/io/pointer"

type bundleKind = uint8

const (
	bundleLabel bundleKind = iota
	bundleButton
	bundleTextInput
	bundleColorPalette
)

func makeColorPalette(systems systems) entity {
	e := newEntity()

	graphics := graphicsComponent{
		kind:       bundleColorPalette,
		isDisabled: true,
	}
	systems.graphics.addComponent(e, graphics)

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{0, 0}}
	systems.bBoxes.addComponent(e, boundingBox)

	return e
}

func makeLabel(
	systems systems,
	width, height int,
	text string, textColor colorID,
	bgColor colorID,
) entity {
	e := newEntity()

	graphics := graphicsComponent{
		text:       text,
		kind:       bundleLabel,
		textColor:  textColor,
		textColors: [8]colorID{textColor, textColor},
		bgColor:    bgColor,
	}
	systems.graphics.addComponent(e, graphics)

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{width, height}}
	systems.bBoxes.addComponent(e, boundingBox)

	return e
}

func makeButton(
	systems systems,
	text string,
	width, height int,
	colorDisabled, colorIdle, colorPressed, colorHovered colorID,
	textColorDisabled, textColorIdle, textColorPressed, textColorHovered colorID,
) entity {
	e := newEntity()

	state := stateComponent{kind: bundleButton, state: 0}
	systems.states.addComponent(e, state)

	boundingBox := boundingBoxComponent{size: [2]int{width, height}}
	systems.bBoxes.addComponent(e, boundingBox)

	interactable := interactableComponent{
		tag:        e,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	systems.interactables.addComponent(e, interactable)

	colors := [8]colorID{colorDisabled, colorIdle, colorPressed, colorHovered}
	textColors := [8]colorID{textColorDisabled, textColorIdle, textColorPressed, textColorHovered}
	graphics := graphicsComponent{
		text:       text,
		textColor:  colorWhite,
		textColors: textColors,
		bgColors:   colors,
		bgColor:    colors[btnStateIdle],
		kind:       bundleButton,
	}
	systems.graphics.addComponent(e, graphics)

	return e
}

func makeTextInput(
	systems systems,
	width, height int,
	placeholderText string,
	textColorDisabled, textColorIdle, textColorHovered, textColorFocused colorID,
	colorDisabled, colorIdle, colorHovered, colorFocused colorID,
) entity {
	e := newEntity()

	state := stateComponent{kind: bundleTextInput, state: 0}
	systems.states.addComponent(e, state)

	boundingBox := boundingBoxComponent{size: [2]int{width, height}}
	systems.bBoxes.addComponent(e, boundingBox)

	interactable := interactableComponent{
		tag:        e,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	systems.interactables.addComponent(e, interactable)

	colors := [8]colorID{colorDisabled, colorIdle, colorHovered, colorFocused}
	textColors := [8]colorID{textColorDisabled, textColorIdle, textColorHovered, textColorFocused}
	graphics := graphicsComponent{
		text:            placeholderText,
		placeholderText: placeholderText,
		textColor:       colorWhite,
		textColors:      textColors,
		bgColors:        colors,
		bgColor:         colors[btnStateIdle],
		borderColor:     colorBlack,
		border:          border{2, 2, 2, 2},
		kind:            bundleTextInput,
	}
	systems.graphics.addComponent(e, graphics)

	return e
}
