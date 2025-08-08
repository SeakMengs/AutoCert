package repository

import (
	"context"
	"errors"

	"github.com/SeakMengs/AutoCert/internal/auth"
	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type JWTRepository struct {
	*baseRepository
	user *UserRepository
}

func (jr JWTRepository) GenRefreshAndAccessToken(ctx context.Context, tx *gorm.DB, user model.User) (*string, *string, error) {
	jr.logger.Debugf("Generate refresh and access token for userId: %s \n", user.ID)

	refreshToken, accessToken, err := jr.jwtService.GenerateRefreshAndAccessToken(auth.JWTPayload{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		ProfileURL: user.ProfileURL,
	})
	if err != nil {
		return nil, nil, err
	}

	db := jr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Token{}).Create(&model.Token{
		RefreshToken: *refreshToken,
		AccessToken:  *accessToken,
		CanAccess:    true,
		CanRefresh:   true,
		UserID:       user.ID,
	}).Error; err != nil {
		return nil, nil, err
	}

	return refreshToken, accessToken, err
}

func (jr JWTRepository) GetTokenByRefreshToken(ctx context.Context, tx *gorm.DB, refreshToken string) (*model.Token, error) {
	jr.logger.Debugf("Get token by  refresh token: %s \n", refreshToken)

	db := jr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var token model.Token

	if err := db.WithContext(ctx).Model(&model.Token{}).Where(model.Token{
		RefreshToken: refreshToken,
	}).First(&token).Error; err != nil {
		return nil, err
	}

	return &token, nil
}

/*
 * Refresh token by deleting the old token and generating new refresh and access token
 */
func (jr JWTRepository) RefreshToken(ctx context.Context, tx *gorm.DB, refreshToken string) (*string, *string, error) {
	jr.logger.Debugf("Refresh token: %s \n", refreshToken)

	db := jr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var newRefreshToken, newAccessToken *string

	txErr := jr.withTx(db, func(tx2 *gorm.DB) error {
		token, err := jr.GetTokenByRefreshToken(ctx, tx2, refreshToken)
		if err != nil {
			return err
		}

		if !token.CanRefresh {
			return errors.New("token is valid but cannot be refreshed")
		}

		var user model.User
		if err := tx2.WithContext(ctx).Model(&model.User{}).Where(model.User{
			BaseModel: model.BaseModel{
				ID: token.UserID,
			},
		}).First(&user).Error; err != nil {
			return err
		}

		newRefreshToken, newAccessToken, err = jr.jwtService.GenerateRefreshAndAccessToken(auth.JWTPayload{
			ID:         user.ID,
			Email:      user.Email,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			ProfileURL: user.ProfileURL,
		})
		if err != nil {
			return err
		}

		if newRefreshToken == nil || newAccessToken == nil {
			return errors.New("failed to generate refresh and access token")
		}

		// Update the new token to the database
		if err := tx2.WithContext(ctx).Model(&model.Token{}).Select("refresh_token", "access_token", "can_acess", "can_refresh", "user_id").Where(model.Token{
			RefreshToken: refreshToken,
		}).Updates(model.Token{
			RefreshToken: *newRefreshToken,
			AccessToken:  *newAccessToken,
			CanAccess:    true,
			CanRefresh:   true,
			UserID:       user.ID,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	// Log tx error
	if txErr != nil {
		jr.logger.Debugf("Refresh token, Transaction error: %v \n", txErr)
	}

	return newRefreshToken, newAccessToken, txErr
}

func (jr JWTRepository) DeleteToken(ctx context.Context, tx *gorm.DB, refreshToken string) error {
	jr.logger.Debugf("Delete token using refresh token: %s \n", refreshToken)

	db := jr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Token{}).Where(model.Token{
		RefreshToken: refreshToken,
	}).Delete(&model.Token{}).Error; err != nil {
		return err
	}

	return nil
}
