package filestorage

import (
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioClient(cfg *config.MinioConfig) (*minio.Client, error) {
	return minio.New(cfg.ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ACCESS_KEY, cfg.SECRET_KEY, ""),
		Secure: cfg.USE_SSL,
		Region: "us-east-1",
	})
}
