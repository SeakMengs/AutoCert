package appcontext

import (
	"github.com/SeakMengs/go-api-boilerplate/internal/auth"
	"github.com/SeakMengs/go-api-boilerplate/internal/config"
	"github.com/SeakMengs/go-api-boilerplate/internal/mailer"
	"github.com/SeakMengs/go-api-boilerplate/internal/repository"
	"go.uber.org/zap"
)

// Application contains core dependencies for the app.
type Application struct {
	// Config holds application settings provided from .env file.
	Config *config.Config

	// Logger lol....
	Logger *zap.SugaredLogger

	// Repository provides access to data storage operations.
	Repository *repository.Repository

	// Mailer handles email-sending functions.
	Mailer mailer.Client

	// JWTService manages JWT operations for authentication such as generate, verify, refresh token.
	JWTService auth.JWTInterface
}
