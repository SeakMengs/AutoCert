package util

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

func ReadBearerToken(ctx *gin.Context) (string, error) {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		return "", errors.New("No bearer token specified.")
	}

	headerParts := strings.SplitN(header, " ", 2)

	if len(headerParts) != 2 {
		return "", errors.New("Wrong bearer token format")
	}

	token := headerParts[1]
	if token == "" {
		return "", errors.New("Bearer token empty")
	}

	if !strings.EqualFold(headerParts[0], "BEARER") {
		return "", errors.New("Invalid bearer type")
	}

	return token, nil
}
