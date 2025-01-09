package auth

import (
	"log"
	"testing"

	"github.com/SeakMengs/go-api-boilerplate/internal/config"
	"github.com/joho/godotenv"
)

// Perform token generation and verify the generated token to ensure VerifyJwtToken is correct
func TestJWT(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}

	cfg := config.GetConfig()

	jwtService := NewJwt(cfg.Auth, nil)
	refreshToken, accessToken, err := jwtService.GenerateRefreshAndAccessToken(JWTPayload{
		UserID: "id1234",
		Email:  "test@gmail.com",
	})
	// t.Errorf("Refrestoken: %s, Accestoken: %s, Error: %v", *refreshToken, *accessToken, err)
	//
	if err != nil {
		t.Errorf(
			"An error occurred during refresh token and access token generation. Error: %v", err)
	}

	if err := jwtService.VerifyJwtToken(*refreshToken); err != nil {
		t.Errorf(
			"An error occurred during refresh token verification. Error: %v", err)
	}

	if err := jwtService.VerifyJwtToken(*accessToken); err != nil {
		t.Errorf(
			"An error occurred during access token verification. Error: %v", err)
	}
}
