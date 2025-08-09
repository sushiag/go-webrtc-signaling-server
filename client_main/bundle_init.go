package main

import "gioui.org/io/pointer"

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
	config labelConfig,
) entity {
	e := newEntity()

	graphics := graphicsComponent{
		text:       config.text,
		kind:       bundleLabel,
		textColor:  config.textColor,
		textColors: [8]colorID{config.textColor, config.textColor},
		bgColor:    config.bgColor,
	}
	systems.graphics.addComponent(e, graphics)

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{config.width, config.height}}
	systems.bBoxes.addComponent(e, boundingBox)

	return e
}

func makeButton(
	systems systems,
	config buttonConfig,
) entity {
	e := newEntity()

	state := stateComponent{kind: bundleButton, state: 0}
	systems.states.addComponent(e, state)

	boundingBox := boundingBoxComponent{size: [2]int{config.width, config.height}}
	systems.bBoxes.addComponent(e, boundingBox)

	interactable := interactableComponent{
		tag:        e,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	systems.interactables.addComponent(e, interactable)

	colors := [8]colorID{config.colorDisabled, config.colorIdle, config.colorPressed, config.colorHovered}
	textColors := [8]colorID{config.textColorDisabled, config.textColorIdle, config.textColorPressed, config.textColorHovered}
	graphics := graphicsComponent{
		text:       config.text,
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
	config textInputConfig,
) entity {
	e := newEntity()

	state := stateComponent{kind: bundleTextInput, state: 0}
	systems.states.addComponent(e, state)

	boundingBox := boundingBoxComponent{size: [2]int{config.width, config.height}}
	systems.bBoxes.addComponent(e, boundingBox)

	interactable := interactableComponent{
		tag:        e,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}
	systems.interactables.addComponent(e, interactable)

	colors := [8]colorID{config.colorDisabled, config.colorIdle, config.colorHovered, config.colorFocused}
	textColors := [8]colorID{config.textColorDisabled, config.textColorIdle, config.textColorHovered, config.textColorFocused}
	graphics := graphicsComponent{
		text:            config.placeholderText,
		placeholderText: config.placeholderText,
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
