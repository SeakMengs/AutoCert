package util

import (
	"strings"
	"testing"
)

func TestAddUniqueSuffixToFilename(t *testing.T) {
	filename := "testfile.txt"
	result := AddUniqueSuffixToFilename(filename)

	if !strings.HasPrefix(result, "testfile_") || !strings.HasSuffix(result, ".txt") {
		t.Errorf("Expected filename to have unique suffix, got %s", result)
	}

	if len(result) != len(filename)+UNIQUE_ID_LENGTH+1 {
		t.Errorf("Expected filename length to be %d, got %d. Result %s", len(filename)+UNIQUE_ID_LENGTH+1, len(result), result)
	}
}

func TestRemoveUniqueSuffixFromFilename(t *testing.T) {
	filename := "testfile_1234567890abcdef.txt"
	expected := "testfile.txt"
	result := RemoveUniqueSuffixFromFilename(filename)

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
