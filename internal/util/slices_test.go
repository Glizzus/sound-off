package util

import (
	"testing"
)

func TestFindFirst(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		predicate func(int) bool
		expected  int
		found     bool
	}{
		{
			name:      "Element found",
			slice:     []int{1, 2, 3, 4, 5},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  2,
			found:     true,
		},
		{
			name:      "Element not found",
			slice:     []int{1, 3, 5, 7},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  0,
			found:     false,
		},
		{
			name:      "Empty slice",
			slice:     []int{},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  0,
			found:     false,
		},
		{
			name:      "First element matches",
			slice:     []int{10, 20, 30},
			predicate: func(x int) bool { return x == 10 },
			expected:  10,
			found:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := FindFirst(tt.slice, tt.predicate)
			if result != tt.expected || found != tt.found {
				t.Errorf("FindFirst() = (%v, %v), want (%v, %v)", result, found, tt.expected, tt.found)
			}
		})
	}
}
