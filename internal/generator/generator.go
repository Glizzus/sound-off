package generator

import (
	"github.com/google/uuid"
)

// Generator is an interface that defines a method to generate a new value of type T.
// This can be used to generate unique identifiers, lazily iterate, etc.
type Generator[T any] interface {
	Next() (T, error)
}

// UUIDV4Generator is a generator that produces UUIDv4 strings.
// It implements the Generator interface.
type UUIDV4Generator struct{}

func (g *UUIDV4Generator) Next() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

var _ Generator[string] = &UUIDV4Generator{}
