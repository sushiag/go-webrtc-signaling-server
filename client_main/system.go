package main

import (
	"log"
	"math"
	"reflect"
)

type entity = uint32

type system struct {
	db           *[]systemDBEntry
	states       *[]stateComponent
	statesEntity *[]entity

	bboxes       *[]boundingBoxComponent
	bboxesEntity *[]entity

	interactables       *[]interactableComponent
	interactablesEntity *[]entity

	graphics       *[]graphicsComponent
	graphicsEntity *[]entity
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
	states := make([]stateComponent, 0, initCapacity)
	statesEntity := make([]entity, 0, initCapacity)
	bboxes := make([]boundingBoxComponent, 0, initCapacity)
	bboxesEntity := make([]entity, 0, initCapacity)
	interactables := make([]interactableComponent, 0, initCapacity)
	interactablesEntity := make([]entity, 0, initCapacity)
	graphics := make([]graphicsComponent, 0, initCapacity)
	graphicsEntity := make([]entity, 0, initCapacity)
	return system{
		db:                  &db,
		states:              &states,
		statesEntity:        &statesEntity,
		bboxes:              &bboxes,
		bboxesEntity:        &bboxesEntity,
		interactables:       &interactables,
		interactablesEntity: &interactablesEntity,
		graphics:            &graphics,
		graphicsEntity:      &graphicsEntity,
	}
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
			idx := len(*sys.states)
			if idx > math.MaxUint16 {
				log.Panicf("max number of state components reached")
			}
			row.components[compKindState] = uint16(idx)
			row.flags |= flagState
			*sys.states = append(*sys.states, c)
			*sys.statesEntity = append(*sys.statesEntity, e)
		case boundingBoxComponent:
			idx := len(*sys.bboxes)
			if idx > math.MaxUint16 {
				log.Panicf("max number of bounding box components reached")
			}
			row.components[compKindBBox] = uint16(idx)
			row.flags |= flagBBox
			*sys.bboxes = append(*sys.bboxes, c)
			*sys.bboxesEntity = append(*sys.bboxesEntity, e)
		case interactableComponent:
			idx := len(*sys.interactables)
			if idx > math.MaxUint16 {
				log.Panicf("max number of interactable components reached")
			}
			row.components[compKindInteractable] = uint16(idx)
			row.flags |= flagInteractable
			*sys.interactables = append(*sys.interactables, c)
			*sys.interactablesEntity = append(*sys.interactablesEntity, e)
		case graphicsComponent:
			idx := len(*sys.graphics)
			if idx > math.MaxUint16 {
				log.Panicf("max number of graphics components reached")
			}
			row.components[compKindGraphics] = uint16(idx)
			row.flags |= flagGraphics
			*sys.graphics = append(*sys.graphics, c)
			*sys.graphicsEntity = append(*sys.graphicsEntity, e)
		default:
			log.Panicln("invalid component type:", reflect.TypeOf(comp))
		}
	}

	*sys.db = append(*sys.db, row)

	return e
}
