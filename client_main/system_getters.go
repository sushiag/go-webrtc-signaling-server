package main

import "log"

func (sys system) getStateComponent(e entity) (stateComponent, bool) {
	var comp stateComponent

	row := (*sys.db)[e]

	if row.flags&flagState != 0 {
		comp = (*sys.states)[row.components[compKindState]]
		return comp, true
	} else {
		return comp, false
	}
}

func (sys *system) getStateComponentRef(e entity) *stateComponent {
	var comp *stateComponent

	row := (*sys.db)[e]

	if row.flags&flagState != 0 {
		comp = &(*sys.states)[row.components[compKindState]]
	} else {
		log.Panicln("entity has no state component:", e)
	}

	return comp
}

func (sys system) getBBoxComponent(e entity) boundingBoxComponent {
	var comp boundingBoxComponent

	db := *(sys.db)
	row := db[e]

	if row.flags&flagBBox != 0 {
		comp = (*sys.bboxes)[row.components[compKindBBox]]
	} else {
		log.Panicln("entity has no bounding box component:", e)
	}

	return comp
}

func (sys *system) getBBoxComponentRef(e entity) *boundingBoxComponent {
	var comp *boundingBoxComponent

	db := *(sys.db)
	row := db[e]

	if row.flags&flagBBox != 0 {
		comp = &(*sys.bboxes)[row.components[compKindBBox]]
	} else {
		log.Panicln("entity has no bounding box component:", e)
	}

	return comp
}

func (sys system) getInteractableComponent(e entity) (interactableComponent, bool) {
	var comp interactableComponent

	row := (*sys.db)[e]

	if row.flags&flagInteractable != 0 {
		comp = (*sys.interactables)[row.components[compKindInteractable]]
		return comp, true
	} else {
		return comp, false
	}
}

func (sys *system) tryGetGraphicsComponentRef(e entity) (*graphicsComponent, uint16, bool) {
	db := *(sys.db)
	row := db[e]

	if row.flags&flagGraphics != 0 {
		idx := row.components[compKindGraphics]
		comp := &(*sys.graphics)[idx]
		return comp, idx, true
	} else {
		return nil, 0, false
	}
}
