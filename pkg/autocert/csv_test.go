package autocert

import (
	"reflect"
	"testing"
)

func TestParseCSVToMap(t *testing.T) {
	tests := []struct {
		name     string
		records  [][]string
		expected []map[string]string
	}{
		{
			name: "Basic CSV",
			records: [][]string{
				{"header1", "header2"},
				{"value1", "value2"},
				{"value3", "value4"},
			},
			expected: []map[string]string{
				{"header1": "value1", "header2": "value2"},
				{"header1": "value3", "header2": "value4"},
			},
		},
		{
			name:     "Empty CSV",
			records:  [][]string{},
			expected: []map[string]string{},
		},
		{
			name: "Missing Values",
			records: [][]string{
				{"header1", "header2"},
				{"value1"},
				{"value3", "value4"},
			},
			expected: []map[string]string{
				{"header1": "value1", "header2": ""},
				{"header1": "value3", "header2": "value4"},
			},
		},
		{
			name: "Extra Values",
			records: [][]string{
				{"header1", "header2"},
				{"value1", "value2", "extra"},
				{"value3", "value4"},
			},
			expected: []map[string]string{
				{"header1": "value1", "header2": "value2"},
				{"header1": "value3", "header2": "value4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCSVToMap(tt.records)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
