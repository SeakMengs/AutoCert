package util

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

func GetProjectDirectoryPath(projectId string) string {
	return fmt.Sprintf("projects/%s", projectId)
}

func ToProjectDirectoryPath(projectId string, filename string) string {
	return filepath.Join(GetProjectDirectoryPath(projectId), filepath.Base(filename))
}

func GetGeneratedCertificateDirectoryPath(projectId string) string {
	return GetProjectDirectoryPath(projectId) + "/generated"
}

func ToGeneratedCertificateDirectoryPath(projectId string, filename string) string {
	return filepath.Join(GetGeneratedCertificateDirectoryPath(projectId), filepath.Base(filename))
}

func createBucketIfNotExists(s3 *minio.Client, bucketName string) error {
	exists, err := s3.BucketExists(context.Background(), bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = s3.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

type FileUploadOptions struct {
	// Add a prefix to the file name
	// For example, if the file name is "data.csv" and the prefix is "projects/123",
	// the resulting name will be "projects/123/data.csv"
	DirectoryPath string
	UniquePrefix  bool
	Bucket        string
	S3            *minio.Client
}

func UploadFileToS3ByFileHeader(fileHeader *multipart.FileHeader, fuo *FileUploadOptions) (minio.UploadInfo, error) {
	if err := createBucketIfNotExists(fuo.S3, fuo.Bucket); err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to create bucket: %w", err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileName := prepareFileName(fileHeader.Filename, fuo)

	info, err := fuo.S3.PutObject(
		context.Background(),
		fuo.Bucket,
		fileName,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return info, nil
}

// uploads a file from a local path to S3
func UploadFileToS3ByPath(path string, fuo *FileUploadOptions) (minio.UploadInfo, error) {
	if err := createBucketIfNotExists(fuo.S3, fuo.Bucket); err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to create bucket: %w", err)
	}

	fileName := prepareFileName(filepath.Base(path), fuo)

	contentType, err := detectContentType(path)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	// Upload the file to S3
	info, err := fuo.S3.FPutObject(
		context.Background(),
		fuo.Bucket,
		fileName,
		path,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return info, nil
}

// Generates the final file name with uniqueness and prefix
func prepareFileName(originalName string, fuo *FileUploadOptions) string {
	fileName := originalName

	if fuo != nil {
		if fuo.UniquePrefix {
			fileName = AddUniquePrefixToFileName(originalName)
		}

		if fuo.DirectoryPath != "" {
			fileName = filepath.Join(fuo.DirectoryPath, fileName)
		}
	}

	return fileName
}

// Determines the content type of a file at the given path
func detectContentType(path string) (string, error) {
	// 1) Try extension-based lookup
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		return contentType, nil
	}

	// 2) Fall back to sniffing the first 512 bytes
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for content type detection: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}

	return http.DetectContentType(buf[:n]), nil
}
