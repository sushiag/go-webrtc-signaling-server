package main

import "log"

func (sys *system) getStateComponentRef(e entity) *stateComponent {
	var comp *stateComponent

	db := *(sys.db)
	row := db[e]

	if row.flags&flagState != 0 {
		comp = &(*sys.states)[row.components[compKindState]]
	} else {
		log.Panicln("entity has no state component:", e)
	}

	return comp
}

func (sys *system) getBBoxComponent(e entity) boundingBoxComponent {
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

func (sys *system) tryGetGraphicsComponentRef(e entity) (*graphicsComponent, bool) {
	db := *(sys.db)
	row := db[e]

	if row.flags&flagGraphics != 0 {
		comp := &(*sys.graphics)[row.components[compKindGraphics]]
		return comp, true
	} else {
		return nil, false
	}
}
