package filestorage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	// Use internal client for faster operations
	internalClient *minio.Client
	// For presigned url
	externalClient *minio.Client
}

func NewMinioClient(cfg *config.MinioConfig) (*MinioClient, error) {
	internalClient, err := minio.New(cfg.INTERNAL_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ACCESS_KEY, cfg.SECRET_KEY, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	externalClient, err := minio.New(cfg.ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ACCESS_KEY, cfg.SECRET_KEY, ""),
		Secure: cfg.USE_SSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinioClient{
		internalClient: internalClient,
		externalClient: externalClient,
	}, nil
}

func (m *MinioClient) InternalClient() *minio.Client {
	return m.internalClient
}

func (m *MinioClient) ExternalClient() *minio.Client {
	return m.externalClient
}

func (m *MinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return m.internalClient.BucketExists(ctx, bucketName)
}

func (m *MinioClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error) {
	return m.internalClient.MakeBucket(ctx, bucketName, opts)
}

func (mc *MinioClient) FPutObject(ctx context.Context, bucketName string, objectName string, filePath string, opts minio.PutObjectOptions) (info minio.UploadInfo, err error) {
	return mc.internalClient.FPutObject(ctx, bucketName, objectName, filePath, opts)
}

func (mc *MinioClient) PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error) {
	return mc.internalClient.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

func (mc *MinioClient) FGetObject(ctx context.Context, bucketName string, objectName string, filePath string, opts minio.GetObjectOptions) error {
	return mc.internalClient.FGetObject(ctx, bucketName, objectName, filePath, opts)
}

func (mc *MinioClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return mc.internalClient.ListObjects(ctx, bucketName, opts)
}

func (mc *MinioClient) RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error {
	return mc.internalClient.RemoveObject(ctx, bucketName, objectName, opts)
}

func (mc *MinioClient) PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error) {
	return mc.externalClient.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
}
