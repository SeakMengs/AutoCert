package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type OAuthProviderRepository struct {
	*baseRepository
}

// Create new oauth or update existing oauth provider accessToken by provider user id
func (opr OAuthProviderRepository) CreateOrUpdateByProviderUserId(ctx context.Context, tx *gorm.DB, newOAuthProvider model.OAuthProvider) error {
	opr.logger.Debugf("Create or update OAuth provider with data: %v \n", newOAuthProvider)

	db := opr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	// Assign mean it will create or update regardless of whether record is found or not
	// It check based on where condition
	if err := db.WithContext(ctx).Model(&model.OAuthProvider{}).Where(&model.OAuthProvider{ProviderUserId: newOAuthProvider.ProviderUserId}).Assign(model.OAuthProvider{
		ProviderType:   newOAuthProvider.ProviderType,
		ProviderUserId: newOAuthProvider.ProviderUserId,
		AccessToken:    newOAuthProvider.AccessToken,
		UserID:         newOAuthProvider.UserID,
	}).FirstOrCreate(&newOAuthProvider).Error; err != nil {
		return err
	}

	return nil
}
