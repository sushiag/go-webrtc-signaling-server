package main

import (
	"log"
	"math"
	"reflect"
)

type entity = uint32

type system struct {
	db            *[]systemDBEntry
	states        *components[stateComponent]
	bboxes        *components[boundingBoxComponent]
	interactables *components[interactableComponent]
	graphics      *components[graphicsComponent]
}

type components[T any] struct {
	comps      []T
	compEntity []entity
}

func (c *components[T]) reset() {
	c.comps = c.comps[:0]
	c.compEntity = c.compEntity[:0]
}

func (c components[T]) len() int {
	return len(c.comps)
}

func (c *components[T]) append(e entity, comp T) {
	c.comps = append(c.comps, comp)
	c.compEntity = append(c.compEntity, e)
}

func newComponents[T any](initialCapacity int) components[T] {
	return components[T]{
		comps:      make([]T, 0, initialCapacity),
		compEntity: make([]entity, 0, initialCapacity),
	}
}

func (comps components[T]) getEntity(idx int) entity {
	return comps.compEntity[idx]
}

type systemDBEntry struct {
	components [7]uint16
	flags      uint8
}

type systemFlag = uint8
type componentKind = uint8

const (
	flagState systemFlag = 1 << iota
	flagBBox
	flagInteractable
	flagGraphics
)

const (
	compKindState componentKind = iota
	compKindBBox
	compKindInteractable
	compKindGraphics
)

func newSystem() system {
	const initCapacity = 8
	db := make([]systemDBEntry, 0, initCapacity)
	states := newComponents[stateComponent](initCapacity)
	bboxes := newComponents[boundingBoxComponent](initCapacity)
	interactables := newComponents[interactableComponent](initCapacity)
	graphics := newComponents[graphicsComponent](initCapacity)
	return system{
		db:            &db,
		states:        &states,
		bboxes:        &bboxes,
		interactables: &interactables,
		graphics:      &graphics,
	}
}

func (sys *system) reset() {
	*sys.db = (*sys.db)[:0]
	sys.states.reset()
	sys.bboxes.reset()
	sys.interactables.reset()
	sys.graphics.reset()
}

func (sys *system) nextEntity() entity {
	return entity(len(*sys.db))
}

func (sys *system) newEntity(components ...any) entity {
	e := entity(len(*sys.db))

	row := systemDBEntry{}

	for _, comp := range components {
		switch c := comp.(type) {
		case stateComponent:
			idx := sys.states.len()
			if idx > math.MaxUint16 {
				log.Panicf("max number of state components reached")
			}
			row.components[compKindState] = uint16(idx)
			row.flags |= flagState
			sys.states.append(e, c)
		case boundingBoxComponent:
			idx := sys.bboxes.len()
			if idx > math.MaxUint16 {
				log.Panicf("max number of bounding box components reached")
			}
			row.components[compKindBBox] = uint16(idx)
			row.flags |= flagBBox
			sys.bboxes.append(e, c)
		case interactableComponent:
			idx := sys.interactables.len()
			if idx > math.MaxUint16 {
				log.Panicf("max number of interactable components reached")
			}
			row.components[compKindInteractable] = uint16(idx)
			row.flags |= flagInteractable
			sys.interactables.append(e, c)
		case graphicsComponent:
			idx := sys.graphics.len()
			if idx > math.MaxUint16 {
				log.Panicf("max number of graphics components reached")
			}
			row.components[compKindGraphics] = uint16(idx)
			row.flags |= flagGraphics
			sys.graphics.append(e, c)
		default:
			log.Panicln("invalid component type:", reflect.TypeOf(comp))
		}
	}

	*sys.db = append(*sys.db, row)

	return e
}
