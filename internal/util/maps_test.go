package util_test

import (
	"testing"

	"github.com/glizzus/sound-off/internal/util"
)

func TestGetOne(t *testing.T) {
	tc := []struct {
		name     string
		input    map[string]string
		expected string
		err      bool
	}{
		{
			name:     "single element",
			input:    map[string]string{"key1": "value1"},
			expected: "value1",
			err:      false,
		},
		{
			name:     "multiple elements",
			input:    map[string]string{"key1": "value1", "key2": "value2"},
			expected: "",
			err:      true,
		},
		{
			name:     "no elements",
			input:    map[string]string{},
			expected: "",
			err:      true,
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			result, err := util.GetOne(test.input)
			if test.err {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != test.expected {
					t.Errorf("expected %v, got %v", test.expected, result)
				}
			}
		})
	}
}
