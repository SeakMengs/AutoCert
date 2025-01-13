package auth

import (
	"errors"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/util"
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
	// For unit test
	if logger == nil {
		logger = util.NewLogger()
	}

	return &JWT{
		jwtSecret: cfg.JWT_SECRET,
		logger:    logger,
	}
}

type JWTPayload struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type JWTClaims struct {
	User JWTPayload `json:"user"`
	IAT  int64      `json:"iat"`
	EXP  int64      `json:"exp"`
}

// Return refreshToken, accessToken, error
func (j JWT) GenerateRefreshAndAccessToken(payload JWTPayload) (*string, *string, error) {
	j.logger.Debugf("Generate refresh and access token with payload: %v", payload)

	// Create refresh token with 7-day expiration
	refreshClaims := jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refresh.SignedString([]byte(j.jwtSecret))
	if err != nil {
		return nil, nil, err
	}

	// Create access token with 5-minute expiration
	accessClaims := jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
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

	// Directly map claims to JWTClaims
	user, ok := claims["user"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid token: user field is missing or malformed")
	}

	return &JWTClaims{
		User: JWTPayload{
			ID:        user["id"].(string),
			Email:     user["email"].(string),
			FirstName: user["firstName"].(string),
			LastName:  user["lastName"].(string),
		},
		IAT: int64(claims["iat"].(float64)),
		EXP: int64(claims["exp"].(float64)),
	}, nil
}
