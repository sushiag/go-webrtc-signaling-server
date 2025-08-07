package main

type entity = uint32

type entityKind = uint8

const (
	entityKindColorPalette entityKind = iota
	entityKindLabel
	entityKindButton
)

// TODO: we will need to turn this into an atomic if we start making entities on
// separate threads but here is fine for now
var entityIDCounter = entity(0)

func newEntity() entity {
	id := entityIDCounter
	entityIDCounter += 1
	return id
}

type system[T any] struct {
	// Component data
	components []T
	// Describes: component index -> entity
	compEntity []entity
	// Describes: entity -> component index
	entityComp map[entity]int
}

func newSystem[T any]() *system[T] {
	const defaultInitCapacity = 32
	return &system[T]{
		make([]T, 0, defaultInitCapacity),
		make([]entity, 0, defaultInitCapacity),
		make(map[entity]int, defaultInitCapacity),
	}
}

// Adds a new component
func (s *system[T]) addComponent(e entity, component T) {
	s.entityComp[e] = len(s.components)
	s.components = append(s.components, component)
	s.compEntity = append(s.compEntity, e)
}

// Tries to get the component for the given entity
func (s system[T]) getComponent(entity entity) (T, int, bool) {
	var component T

	if idx, ok := s.entityComp[entity]; ok {
		component = s.components[idx]
		return component, idx, true
	} else {
		return component, 0, false
	}
}

// Updates the component at the given index with the given copy
//
// Panics: if given an index that doesn't exist
func (s system[T]) updateComponent(idx int, component T) {
	s.components[idx] = component
}

// Tries to get the entity for the given component index
//
// Panics: if given an index that doesn't exist
func (s system[T]) getEntity(idx int) entity {
	return s.compEntity[idx]
}
