package appcontext

import (
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	filestorage "github.com/SeakMengs/AutoCert/internal/file_storage"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"go.uber.org/zap"
)

type Application struct {
	Config *config.Config

	Logger *zap.SugaredLogger

	Repository *repository.Repository

	// manages JWT operations for authentication such as generate, verify, refresh token.
	JWTService auth.JWTInterface

	S3 *filestorage.MinioClient

	Queue *queue.RabbitMQ
}
