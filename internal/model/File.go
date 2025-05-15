package model

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
)

type File struct {
	BaseModel
	FileName       string `gorm:"type:text;not null" json:"fileName" form:"fileName" binding:"required"`
	UniqueFileName string `gorm:"type:text;not null;uniqueIndex" json:"uniqueFileName" form:"uniqueFileName" binding:"required"`
	BucketName     string `gorm:"type:text;not null" json:"bucketName" form:"bucketName" binding:"required"`
	Size           int64  `gorm:"type:bigint;not null" json:"size" form:"size" binding:"required"`
}

func (f File) TableName() string {
	return "files"
}

func (f File) ToPresignedUrl(ctx context.Context, s3 *minio.Client) (string, error) {
	if f.BucketName == "" || f.UniqueFileName == "" {
		return "", errors.New("bucket name and unique file name cannot be empty")
	}

	// Generate a presigned URL for the file
	presignedURL, err := s3.PresignedGetObject(
		ctx,
		f.BucketName,
		f.UniqueFileName,
		// 60min expiration time
		time.Minute*60,
		nil,
	)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (f File) DownloadToLocal(ctx context.Context, s3 *minio.Client, localPath string) error {
	if f.BucketName == "" || f.UniqueFileName == "" || localPath == "" {
		return fmt.Errorf("bucket name, unique file name, and local path cannot be empty: bucket=%s, uniqueFileName=%s, localPath=%s", f.BucketName, f.UniqueFileName, localPath)
	}

	err := s3.FGetObject(ctx, f.BucketName, f.UniqueFileName, localPath, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (f File) Delete(ctx context.Context, s3 *minio.Client) error {
	if f.BucketName == "" || f.UniqueFileName == "" {
		return errors.New("bucket name and unique file name cannot be empty")
	}

	if err := s3.RemoveObject(ctx, f.BucketName, f.UniqueFileName, minio.RemoveObjectOptions{}); err != nil {
		return err
	}

	return nil
}

func (f File) ToBaseFilename() string {
	return filepath.Base(f.FileName)
}

func (f File) ToBaseUniqueFilename() string {
	return filepath.Base(f.UniqueFileName)
}
