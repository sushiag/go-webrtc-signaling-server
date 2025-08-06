package main

type entity = uint32

var entityIDCounter = entity(0)

func newEntity() entity {
	id := entityIDCounter
	entityIDCounter += 1
	return id
}

type entityKind = uint8

const (
	entityKindColorPalette entityKind = iota
	entityKindLabel
	entityKindButton
)
