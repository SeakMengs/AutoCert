package controller

import (
	"context"
	"encoding/json"
	"errors"
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
		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	url := oc.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	oc.app.Logger.Infof("OAuth: Google, Redirect to: %s", url)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (oc OAuthController) getGoogleUserInfo(code string) (*GoogleUser, error) {
	oc.app.Logger.Debug("OAuth: Google, Get user info logic")

	// Exchange the authorization code for an access token
	token, err := oc.googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		oc.app.Logger.Debugf("OAuth: Google, Failed to exchange code for token. Error: %v", err)
		return nil, err
	}

	// Use the access token to fetch user info
	client := oc.googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		oc.app.Logger.Debugf("OAuth: Google, Failed to get user info. Error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo GoogleUser
	userInfo.AccessToken = token.AccessToken

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		oc.app.Logger.Debug("OAuth: Google, Failed to decode user info. Error: %v", err)
		return nil, err
	}

	oc.app.Logger.Debugf("OAuth: Google, User info: %v", userInfo)

	return &userInfo, nil
}

func (oc OAuthController) ContinueWithGoogleCallback(ctx *gin.Context) {
	oc.app.Logger.Debug("OAuth: Google callback logic")

	code := ctx.Query("code")
	userInfo, err := oc.getGoogleUserInfo(code)
	if err != nil {
		oc.app.Logger.Debug("OAuth: Google callback, Failed to get user info. Error: %v", err)

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	tx := oc.app.Repository.DB.Begin()
	defer tx.Commit()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to continue with Google", util.GenerateErrorMessages(errors.New("failed to continue with Google")), nil)
			return
		}
	}()

	oc.app.Logger.Debugf("OAuth: Google callback, Transaction begin")

	// If new user, create account, else do nothing
	user, err := oc.app.Repository.User.CreateOrUpdateByEmail(ctx, tx, model.User{
		Email:      userInfo.Email,
		FirstName:  userInfo.GivenName,
		LastName:   userInfo.Name,
		ProfileURL: userInfo.Picture,
	})
	if err != nil {
		tx.Rollback()
		oc.app.Logger.Debugf("OAuth: Google callback, Failed to create or update user. Error: %v", err)

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err), nil)
		return
	}

	// Creare or update oauth provider such that we can store the access token
	err = oc.app.Repository.OAuthProvider.CreateOrUpdateByProviderUserId(ctx, tx, model.OAuthProvider{
		ProviderUserId: userInfo.ID,
		ProviderType:   constant.OAUTH_PROVIDER_GOOGLE,
		AccessToken:    userInfo.AccessToken,
		UserID:         user.ID,
	})
	if err != nil {
		tx.Rollback()
		oc.app.Logger.Debug("OAuth: Google callback, Failed to create or update oauth provider. Error: %v", err)

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err), nil)
		return
	}

	refreshToken, accessToken, err := oc.app.Repository.JWT.GenRefreshAndAccessToken(ctx, tx, *user)
	if err != nil {
		tx.Rollback()
		oc.app.Logger.Debug("OAuth: Google callback, Failed to generate refresh and access token. Error: %v", err)

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err), nil)
		return
	}

	if refreshToken == nil || accessToken == nil {
		tx.Rollback()
		oc.app.Logger.Debug("OAuth: Google callback, Failed to generate refresh and access token")

		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(errors.New("failed to generate refresh and access token")), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"refreshToken": refreshToken,
		"accessToken":  accessToken,
	})
}
