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
	VerifyJwtToken(token string) error
	RefreshAccessToken(accessToken string) (*string, *string, error)
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
	UserID string `json:"userId"`
	Email  string `json:"email"`
}

// Return refreshToken, accessToken, error
// TODO: implement databse check to get user
func (j JWT) GenerateRefreshAndAccessToken(payload JWTPayload) (*string, *string, error) {
	j.logger.Debugf("Generate refresh and access token with payload: %v", payload)

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		// update refresh token expire date here | 7 days
		"exp": time.Now().Add((time.Hour * 24) * 7).Unix(),
	})

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": payload,
		"iat":  time.Now().Unix(),
		// update access token expire date here | 5 minutes
		"exp": time.Now().Add(time.Minute * 5).Unix(),
	})

	refreshToken, refreshErr := refresh.SignedString([]byte(j.jwtSecret))
	if refreshErr != nil {
		return nil, nil, refreshErr
	}

	accessToken, accessErr := access.SignedString([]byte(j.jwtSecret))
	if accessErr != nil {
		return nil, nil, accessErr
	}

	return &refreshToken, &accessToken, nil
}

func (j JWT) VerifyJwtToken(token string) error {
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.jwtSecret), nil
	})
	if err != nil {
		j.logger.Debugf("Failed to verify jwt token. Error: %v", err)
		return err
	}

	if !parsedToken.Valid {
		j.logger.Debug("Jwt token is not valid")
		return errors.New("Jwt token is not valid")
	}

	return nil
}

func (j JWT) RefreshAccessToken(accessToken string) (*string, *string, error) {
	if err := j.VerifyJwtToken(accessToken); err != nil {
		return nil, nil, err
	}
	// TODO: implement

	return nil, nil, nil
}
