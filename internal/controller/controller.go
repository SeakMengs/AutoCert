package controller

import (
	appcontext "github.com/SeakMengs/AutoCert/internal/app_context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type baseController struct {
	app *appcontext.Application
}

type Controller struct {
	User  *UserController
	Index *IndexController
	Auth  *AuthController
	OAuth *OAuthController
	File  *FileController
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
		User:  &UserController{baseController: bc},
		Index: &IndexController{baseController: bc},
		Auth:  &AuthController{baseController: bc},
		OAuth: &OAuthController{baseController: bc, googleOAuthConfig: googleOAuthConfig},
		File:  &FileController{baseController: bc},
	}
}
