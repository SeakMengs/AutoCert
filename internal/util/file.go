package util

import (
	"fmt"
	"os"
	"time"
)

// Example output for "ex.txt": "21313123123_ex.txt"
func AddUniquePrefixToFileName(fileName string) string {
	uniquePrefix := fmt.Sprintf("%d", time.Now().UnixNano())
	return fmt.Sprintf("%s_%s", uniquePrefix, fileName)
}

func GetTempDir() string {
	return fmt.Sprintf("%s/autocert", os.TempDir())
}

func CreateTemp(pattern string) (*os.File, error) {
	tempDir := GetTempDir()
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	return os.CreateTemp(tempDir, pattern)
}

func MkdirTemp(pattern string) (string, error) {
	tempDir := GetTempDir()
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	dir, err := os.MkdirTemp(tempDir, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory with pattern %s: %w", pattern, err)
	}
	return dir, nil
}
