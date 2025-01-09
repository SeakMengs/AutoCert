package repository

import (
	"context"

	"github.com/SeakMengs/go-api-boilerplate/internal/auth"
	constant "github.com/SeakMengs/go-api-boilerplate/internal/constant"
	"github.com/SeakMengs/go-api-boilerplate/internal/model"
	"gorm.io/gorm"
)

type JWTRepository struct {
	*baseRepository
	user *UserRepository
}

func (jr JWTRepository) GenRefreshAndAccessToken(ctx context.Context, tx *gorm.DB, user model.User) (*string, *string, error) {
	jr.logger.Debugf("Generate refresh and access token for userId: %s \n", user.ID)

	refreshToken, accessToken, err := jr.jwtService.GenerateRefreshAndAccessToken(auth.JWTPayload{
		UserID: user.ID,
		Email:  user.Email,
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
