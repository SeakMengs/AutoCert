package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	appcontext "github.com/SeakMengs/AutoCert/internal/app_context"
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type baseController struct {
	app *appcontext.Application
}

type Controller struct {
	User           *UserController
	Index          *IndexController
	Auth           *AuthController
	OAuth          *OAuthController
	File           *FileController
	Project        *ProjectController
	ProjectBuilder *ProjectBuilderController
	Signature      *SignatureController
	Certificate    *CertificateController
}

func newBaseController(app *appcontext.Application) *baseController {
	return &baseController{app: app}
}

func NewController(app *appcontext.Application) *Controller {
	bc := newBaseController(app)

	googleOAuthConfig := &oauth2.Config{
		ClientID:     app.Config.Auth.GoogleOAuthConfig.ClientID,
		ClientSecret: app.Config.Auth.GoogleOAuthConfig.ClientSecret,
		RedirectURL:  app.Config.Auth.GoogleOAuthConfig.RedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	return &Controller{
		User:           &UserController{baseController: bc},
		Index:          &IndexController{baseController: bc},
		Auth:           &AuthController{baseController: bc},
		OAuth:          &OAuthController{baseController: bc, googleOAuthConfig: googleOAuthConfig},
		File:           &FileController{baseController: bc},
		Project:        &ProjectController{baseController: bc},
		ProjectBuilder: &ProjectBuilderController{baseController: bc},
		Signature:      &SignatureController{baseController: bc},
		Certificate:    &CertificateController{baseController: bc},
	}
}

func (b *baseController) getAuthUser(ctx *gin.Context) (*auth.JWTPayload, error) {
	user, exists := ctx.Get("user")
	if !exists {
		return nil, errors.New("user not found in context")
	}

	jsonUser, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	var authUser *auth.JWTPayload
	err = json.Unmarshal(jsonUser, &authUser)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return authUser, nil
}

func (b *baseController) getProjectRole(ctx *gin.Context, projectId string) ([]constant.ProjectRole, *model.Project, error) {
	user, err := b.getAuthUser(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get auth user: %w", err)
	}

	role, project, err := b.app.Repository.Project.GetRoleOfProject(ctx, nil, projectId, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get project role: %w", err)
	}

	if len(role) == 0 {
		role = []constant.ProjectRole{}
	}

	return role, project, nil
}

type FileUploadOptions struct {
	// Add a prefix to the file name
	// For example, if the file name is "data.csv" and the prefix is "projects/123",
	// the resulting name will be "projects/123/data.csv"
	DirectoryPath string
	UniquePrefix  bool
}

func (b *baseController) uploadFileToS3ByFileHeader(fileHeader *multipart.FileHeader, fuo *FileUploadOptions) (minio.UploadInfo, error) {
	if err := createBucketIfNotExists(b.app.S3, b.app.Config.Minio.BUCKET); err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to create bucket: %w", err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileName := b.prepareFileName(fileHeader.Filename, fuo)

	info, err := b.app.S3.PutObject(
		context.Background(),
		b.app.Config.Minio.BUCKET,
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
func (b *baseController) uploadFileToS3ByPath(path string, fuo *FileUploadOptions) (minio.UploadInfo, error) {
	if err := createBucketIfNotExists(b.app.S3, b.app.Config.Minio.BUCKET); err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to create bucket: %w", err)
	}

	fileName := b.prepareFileName(filepath.Base(path), fuo)

	contentType, err := b.detectContentType(path)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	// Upload the file to S3
	info, err := b.app.S3.FPutObject(
		context.Background(),
		b.app.Config.Minio.BUCKET,
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
func (b *baseController) prepareFileName(originalName string, fuo *FileUploadOptions) string {
	fileName := originalName

	if fuo != nil {
		if fuo.UniquePrefix {
			fileName = util.AddUniquePrefixToFileName(originalName)
		}

		if fuo.DirectoryPath != "" {
			fileName = filepath.Join(fuo.DirectoryPath, fileName)
		}
	}

	return fileName
}

// Determines the content type of a file at the given path
func (b *baseController) detectContentType(path string) (string, error) {
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
