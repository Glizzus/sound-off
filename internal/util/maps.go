package util

import "fmt"

// GetOne returns the single element from a map. If the map is empty, it returns an error.
// If the map has more than one element, it returns an error.
func GetOne[K comparable, T any](m map[K]T) (T, error) {
	var result T
	for _, v := range m {
		result = v
		if len(m) == 1 {
			return result, nil
		}

		return result, fmt.Errorf("multiple elements found")

	}
	var zero T
	return zero, fmt.Errorf("no element found")
}
