package generator

import (
	"github.com/google/uuid"
)

type Generator[T any] interface {
	Next() (T, error)
}

type UUIDV4Generator struct{}

func (g *UUIDV4Generator) Next() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

var _ Generator[string] = &UUIDV4Generator{}
