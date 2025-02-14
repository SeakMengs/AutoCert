package util

import (
	"testing"
)

func TestGenerateNChar(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"Generate 5 characters", 5, false},
		{"Generate 10 characters", 10, false},
		{"Generate 0 characters", 0, false},
		{"Generate negative characters", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateNChar(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateNChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.n {
				t.Errorf("GenerateNChar() got = %v, want length %v", got, tt.n)
			}
		})
	}
}
