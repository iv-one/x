package entity

import (
	"crypto/rand"
	"io"
	"time"
)

// TimeMachine is an interface for time machines
type TimeMachine interface {
	Now() time.Time
}

// EntityType is an enum for entity types
type EntityType string

// String returns the string representation of the entity type
func (e EntityType) String() string {
	return string(e)
}

// ID is an interface for entity IDs.
//
// Eq reports whether both ids are set and have the same Str: an unset id (nil,
// typed nil, or IsNil) equals nothing, itself included, so two missing ids
// never match. Str, Bytes, Eq and IsNil never panic, on a nil receiver either.
type ID interface {
	Str() string
	Bytes() []byte
	Eq(ID) bool
	IsNil() bool
}

// Entity is an interface for entities
type Entity interface {
	ID() ID
	Type() EntityType
	New() Entity
	IsNil() bool
	Update(TimeMachine)
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// NIL ID
type nilID struct{}

func (n *nilID) Str() string {
	return ""
}

func (n *nilID) Bytes() []byte {
	return []byte{}
}

func (n *nilID) Eq(_ ID) bool {
	return false
}

func (n *nilID) IsNil() bool {
	return n == nil
}

// NilID is a nil ID
var NilID ID = &nilID{}

// RawID is a raw ID used on low level
type RawID []byte

// Str returns the string representation of the raw ID
func (r RawID) Str() string {
	return string(r)
}

// Bytes returns the byte representation of the raw ID
func (r RawID) Bytes() []byte {
	return []byte(r)
}

// Eq returns true if the raw ID is equal to the given ID
func (r RawID) Eq(id ID) bool {
	return len(r) > 0 && id != nil && !id.IsNil() && r.Str() == id.Str()
}

// IsNil returns true if the raw ID is nil
func (r RawID) IsNil() bool {
	return len(r) == 0
}

// NewRawID creates a new raw ID from the given byte slice
func NewRawID(b []byte) RawID {
	raw := make([]byte, len(b))
	copy(raw, b)
	return RawID(raw)
}

// NewRandomRawID creates a new random raw ID
func NewRandomRawID() RawID {
	bytes := make([]byte, 4)

	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		bytes = []byte{0, 0, 0, 0}
	}

	return NewRawID(bytes)
}

// EntityList is an interface for entity lists
type EntityList interface {
	List() []Entity
	IsNil() bool
}

type elist struct {
	list []Entity
}

func (e *elist) List() []Entity {
	return e.list
}

func (e *elist) IsNil() bool {
	return e == nil || e.list == nil
}

// EList creates a new entity list from the given entities
func EList[T Entity](xs []T) EntityList {
	if xs == nil {
		xs = []T{}
	}
	ys := make([]Entity, len(xs))
	for i, x := range xs {
		ys[i] = x
	}
	return &elist{list: ys}
}

// FromMap creates a new entity list from the given map
func FromMap[T Entity](m map[string]T) EntityList {
	xs := make([]T, 0, len(m))
	for _, x := range m {
		xs = append(xs, x)
	}
	return EList[T](xs)
}

// ToMap creates a new map from the given entities
func ToMap[T Entity](xs ...T) map[string]T {
	m := make(map[string]T)
	for _, x := range xs {
		m[x.ID().Str()] = x
	}
	return m
}

// FromMapIDX creates a new list from the given map
func FromMapIDX[T ID](m map[string]T) []T {
	ids := make([]T, 0, len(m))
	for _, id := range m {
		ids = append(ids, id)
	}
	return ids
}

// FromMapID creates a new list from the given map
func FromMapID[T ID](m map[string]T) []ID {
	ids := make([]ID, 0, len(m))
	for _, id := range m {
		ids = append(ids, id)
	}
	return ids
}

// ToList creates a new list from the given IDs
func ToList[T ID](xs []T) []ID {
	ids := make([]ID, 0, len(xs))
	for _, x := range xs {
		ids = append(ids, x)
	}
	return ids
}
