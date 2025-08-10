package main

import "gioui.org/io/pointer"

func makeColorPalette(sys system) entity {
	graphicsComp := graphicsComponent{
		kind:       bundleColorPalette,
		isDisabled: true,
	}

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{0, 0}}

	return sys.newEntity(graphicsComp, boundingBox)
}

func makeLabel(
	sys system,
	config labelConfig,
) entity {
	graphics := graphicsComponent{
		text:       config.text,
		kind:       bundleLabel,
		textColor:  config.textColor,
		textColors: [8]colorID{config.textColor, config.textColor},
		bgColor:    config.bgColor,
	}

	boundingBox := boundingBoxComponent{[2]int{0, 0}, [2]int{config.width, config.height}}

	return sys.newEntity(graphics, boundingBox)
}

func makeButton(
	sys system,
	config buttonConfig,
) entity {
	stateComp := stateComponent{kind: bundleButton, state: config.initState}

	boundingBox := boundingBoxComponent{size: [2]int{config.width, config.height}}

	interactableComp := interactableComponent{
		tag:        sys.nextEntity(),
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}

	colors := [8]colorID{config.colorDisabled, config.colorIdle, config.colorPressed, config.colorHovered}
	textColors := [8]colorID{config.textColorDisabled, config.textColorIdle, config.textColorPressed, config.textColorHovered}
	graphicsComp := graphicsComponent{
		text:       config.text,
		textColor:  colorWhite,
		textColors: textColors,
		bgColors:   colors,
		bgColor:    colors[btnStateIdle],
		kind:       bundleButton,
	}

	return sys.newEntity(stateComp, boundingBox, interactableComp, graphicsComp)
}

func makeTextInput(
	sys system,
	config textInputConfig,
) entity {
	stateComp := stateComponent{kind: bundleTextInput, state: config.initState}

	bboxComp := boundingBoxComponent{size: [2]int{config.width, config.height}}

	interactableComp := interactableComponent{
		tag:        sys.nextEntity(),
		focusable:  true,
		ptrEvFlags: pointer.Enter | pointer.Leave | pointer.Press | pointer.Release,
	}

	colors := [8]colorID{config.colorDisabled, config.colorIdle, config.colorHovered, config.colorFocused}
	textColors := [8]colorID{config.textColorDisabled, config.textColorIdle, config.textColorHovered, config.textColorFocused}
	graphicsComp := graphicsComponent{
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

	return sys.newEntity(stateComp, bboxComp, interactableComp, graphicsComp)
}
