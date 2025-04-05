package util

import (
	"strings"
	"testing"
)

func TestAddUniquePrefixToFileName(t *testing.T) {
	filename := "testfile.txt"
	result := AddUniquePrefixToFileName(filename)

	if !strings.HasSuffix(result, "_testfile.txt") {
		t.Errorf("Expected filename to have unique prefix, got %s", result)
	}

	prefix := strings.Split(result, "_")[0]
	if len(prefix) == 0 {
		t.Errorf("Expected a non-empty unique prefix, got %s", prefix)
	}
}
