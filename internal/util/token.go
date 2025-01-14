package util

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

// Read Authorization header from the request and return the token type and token
func ReadAuthorizationHeader(ctx *gin.Context) (string, string, error) {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		return "", "", errors.New("no authorization header specified")
	}

	headerParts := strings.SplitN(header, " ", 2)
	if len(headerParts) != 2 {
		return "", "", errors.New("wrong authorization header format")
	}

	tokenType := strings.ToUpper(headerParts[0])
	token := headerParts[1]

	if token == "" {
		return "", "", errors.New("token is empty")
	}

	return tokenType, token, nil
}

// Read Bearer token from the request Authorization header and return the token
func ReadBearerToken(ctx *gin.Context) (string, error) {
	tokenType, token, err := ReadAuthorizationHeader(ctx)
	if err != nil {
		return "", err
	}

	if !strings.EqualFold(tokenType, "BEARER") {
		return "", errors.New("invalid token type; expected 'Bearer'")
	}

	return token, nil
}

// Read Refresh token from the request Authorization header and return the token
func ReadRefreshToken(ctx *gin.Context) (string, error) {
	tokenType, token, err := ReadAuthorizationHeader(ctx)
	if err != nil {
		return "", err
	}

	if !strings.EqualFold(tokenType, "REFRESH") {
		return "", errors.New("invalid token type; expected 'Refresh'")
	}

	return token, nil
}
