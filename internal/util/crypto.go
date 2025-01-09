package util

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

func GenerateNChar(n int) (string, error) {
	data := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
