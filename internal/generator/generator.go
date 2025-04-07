package generator

import (
	"github.com/google/uuid"
)

type Generator[T any] interface {
	Next() (T, error)
}

type UUIDGenerator struct{}

func (g *UUIDGenerator) Next() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

var _ Generator[string] = &UUIDGenerator{}
