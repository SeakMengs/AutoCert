package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (b *baseController) uploadFileToS3ByPath(path string) (minio.UploadInfo, error) {
	err := createBucketIfNotExists(b.app.S3, b.app.Config.Minio.BUCKET)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to create bucket: %w", err)
	}

	// get file name
	fileName := filepath.Base(path)

	// Determine the content type of the file
	contentType := "application/octet-stream" // Default content type
	file, err := os.Open(path)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to read file: %w", err)
	}

	contentType = http.DetectContentType(buffer)

	// Upload the file to S3
	info, err := b.app.S3.FPutObject(context.Background(), b.app.Config.Minio.BUCKET, util.AddUniquePrefixToFileName(fileName), path, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return info, nil
}
