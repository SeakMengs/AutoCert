package appcontext

import (
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Application struct {
	Config *config.Config

	Logger *zap.SugaredLogger

	Repository *repository.Repository

	// manages JWT operations for authentication such as generate, verify, refresh token.
	JWTService auth.JWTInterface

	S3 *minio.Client

	Queue *queue.RabbitMQ
}
