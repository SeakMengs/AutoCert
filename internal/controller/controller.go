package controller

import (
	"encoding/json"
	"errors"
	"fmt"

	appcontext "github.com/SeakMengs/AutoCert/internal/app_context"
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/gin-gonic/gin"
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

func (b *baseController) getProjectRole(ctx *gin.Context, projectId string) (*auth.JWTPayload, []constant.ProjectRole, *model.Project, error) {
	user, err := b.getAuthUser(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get auth user: %w", err)
	}

	roles, project, err := b.app.Repository.Project.GetRoleOfProject(ctx, nil, projectId, user)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get project role: %w", err)
	}

	return user, roles, project, nil
}
