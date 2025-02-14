package util

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenerateNChar(n int) (string, error) {
	id, err := gonanoid.New(n)
	if err != nil {
		return "", err
	}
	return id, nil
}
