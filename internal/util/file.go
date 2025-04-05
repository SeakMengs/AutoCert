package util

import (
	"fmt"
	"time"
)

// Example output for "ex.txt": "21313123123_ex.txt"
func AddUniquePrefixToFileName(fileName string) string {
	uniquePrefix := fmt.Sprintf("%d", time.Now().UnixNano())
	return fmt.Sprintf("%s_%s", uniquePrefix, fileName)
}
