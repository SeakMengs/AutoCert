package util

import (
	"fmt"
	"path/filepath"
)

const UNIQUE_ID_LENGTH = 16

// add unique id to filename to make it unique such that it doesn't overwrite existing files
// Example input: filename.txt
// Example output: filename_FpqnQB-UeY3Ulab7.txt
func AddUniqueSuffixToFilename(filename string) string {
	ext := filepath.Ext(filename)
	uniqueId, err := GenerateNChar(UNIQUE_ID_LENGTH)
	if err != nil {
		return filename
	}
	name := filename[:len(filename)-len(ext)]

	return fmt.Sprintf("%s_%s%s", name, uniqueId, ext)
}

// remove unique id from filename
// Example input: filename_FpqnQB-UeY3Ulab7.txt
// Example output: filename.txt
func RemoveUniqueSuffixFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	// -1 to remove the underscore
	return filename[:len(filename)-len(ext)-UNIQUE_ID_LENGTH-1] + ext
}
