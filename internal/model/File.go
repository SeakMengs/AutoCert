package model

import (
	"context"
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
	// Generate a presigned URL for the file
	presignedURL, err := s3.PresignedGetObject(
		ctx,
		f.BucketName,
		f.UniqueFileName,
		// 30min expiration time
		time.Minute*30,
		nil,
	)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
