package auth

import (
	"log"
	"testing"

	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/joho/godotenv"
)

// Perform token generation and verify the generated token to ensure VerifyJwtToken is correct
func TestJWT(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}

	cfg := config.GetConfig()
	logger := util.NewLogger(cfg.ENV)

	jwtService := NewJwt(cfg.Auth, logger)
	refreshToken, accessToken, err := jwtService.GenerateRefreshAndAccessToken(JWTPayload{
		ID:    "id1234",
		Email: "test@gmail.com",
	})

	// t.Errorf("Refrestoken: %s, Accestoken: %s, Error: %v", *refreshToken, *accessToken, err)

	if err != nil {
		t.Errorf(
			"An error occurred during refresh token and access token generation. Error: %v", err)
	}

	if _, err := jwtService.VerifyJwtToken(*refreshToken); err != nil {
		t.Errorf(
			"An error occurred during refresh token verification. Error: %v", err)
	}

	if _, err := jwtService.VerifyJwtToken(*accessToken); err != nil {
		t.Errorf(
			"An error occurred during access token verification. Error: %v", err)
	}
}
