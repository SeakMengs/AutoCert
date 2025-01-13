package controller

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type OAuthController struct {
	*baseController
	googleOAuthConfig *oauth2.Config
}

type GoogleUser struct {
	Email         string `json:"email"`
	GivenName     string `json:"given_name"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	AccessToken   string `json:"-"`
}

func (oc OAuthController) ContinueWithGoogle(ctx *gin.Context) {
	oc.app.Logger.Debug("OAuth: Google logic")

	state, err := util.GenerateNChar(16)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	url := oc.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	oc.app.Logger.Debugf("OAuth: Google, Redirect to: %s", url)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (oc OAuthController) getGoogleUserInfo(code string) (*GoogleUser, error) {
	oc.app.Logger.Debug("OAuth: Google, Get user info logic")

	// Exchange the authorization code for an access token
	token, err := oc.googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to exchange token")
		return nil, err
	}

	// Use the access token to fetch user info
	client := oc.googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to fetch user info")
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo GoogleUser
	userInfo.AccessToken = token.AccessToken

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to decode user info")
		return nil, err
	}

	return &userInfo, nil
}

func (oc OAuthController) ContinueWithGoogleCallback(ctx *gin.Context) {
	oc.app.Logger.Debug("OAuth: Google callback logic")

	code := ctx.Query("code")
	userInfo, err := oc.getGoogleUserInfo(code)
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to get user info")

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	// If new user, create account, else do nothing
	oc.app.Repository.User.CheckDupAndCreate(ctx, nil, model.User{
		Email:     userInfo.Email,
		FirstName: userInfo.GivenName,
		LastName:  userInfo.Name,
	})

	user, err := oc.app.Repository.User.GetByEmail(ctx, nil, userInfo.Email)
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to get user by email")

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	// Creare or update oauth provider such that we can store the access token
	oc.app.Repository.OAuthProvider.CreateOrUpdateByProviderUserId(ctx, nil, model.OAuthProvider{
		ProviderUserId: userInfo.ID,
		ProviderType:   constant.OAUTH_PROVIDER_GOOGLE,
		AccessToken:    userInfo.AccessToken,
		UserID:         user.ID,
	})

	refreshToken, accessToken, err := oc.app.Repository.JWT.GenRefreshAndAccessToken(ctx, nil, *user)
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google, Error: Failed to generate refresh and access token")

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"refreshToken": refreshToken,
		"accessToken":  accessToken,
	})
}
