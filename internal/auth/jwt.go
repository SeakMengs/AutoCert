package auth

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWT struct {
	logger    *zap.SugaredLogger
	jwtSecret string
}

type JWTInterface interface {
	GenerateRefreshAndAccessToken(payload JWTPayload) (*string, *string, error)
	VerifyJwtToken(token string) (*JWTClaims, error)
}

func NewJwt(cfg config.AuthConfig, logger *zap.SugaredLogger) *JWT {
	return &JWT{
		jwtSecret: cfg.JWT_SECRET,
		logger:    logger,
	}
}

type JWTPayload struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	ProfileURL string `json:"profileUrl"`
}

type JWTClaims struct {
	User JWTPayload `json:"user"`
	IAT  int64      `json:"iat"`
	EXP  int64      `json:"exp"`
	Type string     `json:"type"`
}

// Return refreshToken, accessToken, error
func (j JWT) GenerateRefreshAndAccessToken(payload JWTPayload) (*string, *string, error) {
	j.logger.Debugf("Generate refresh and access token with payload: %v", payload)

	// Create refresh token with 7-day expiration
	refreshClaims := jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": constant.JWT_TYPE_REFRESH,
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refresh.SignedString([]byte(j.jwtSecret))
	if err != nil {
		return nil, nil, err
	}

	// Create access token with 15-minute expiration
	accessClaims := jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"type": constant.JWT_TYPE_ACCESS,
	}
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := access.SignedString([]byte(j.jwtSecret))
	if err != nil {
		return nil, nil, err
	}

	return &refreshToken, &accessToken, nil
}

func (j JWT) VerifyJwtToken(token string) (*JWTClaims, error) {
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.jwtSecret), nil
	})
	if err != nil {
		j.logger.Debugf("Failed to verify jwt token. Error: %v", err)
		return nil, err
	}

	if !parsedToken.Valid {
		j.logger.Debug("Jwt token is not valid")
		return nil, errors.New("jwt token is not valid")
	}

	// Marshal claims to JSON so we can unmarshal into a struct later
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		j.logger.Debugf("Failed to marshal claims. Error: %v", err)
		return nil, errors.New("failed to process token claims")
	}

	// Unmarshal into a JWTClaims struct
	var jwtClaims JWTClaims
	err = json.Unmarshal(claimsJSON, &jwtClaims)
	if err != nil {
		j.logger.Debugf("Failed to unmarshal claims to JWTClaims. Error: %v", err)
		return nil, errors.New("invalid token structure")
	}

	return &jwtClaims, nil
}
