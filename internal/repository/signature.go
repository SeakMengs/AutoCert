package repository

import (
	"context"

	"github.com/SeakMengs/AutoCert/internal/auth"
	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type SignatureRepository struct {
	*baseRepository
}

func (sr SignatureRepository) Create(ctx context.Context, tx *gorm.DB, signature *model.Signature) (*model.Signature, error) {
	sr.logger.Debugf("Create signature with data: %v \n", signature)

	db := sr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Signature{}).Create(&signature).Error; err != nil {
		return signature, err
	}

	return signature, nil
}

func (sr SignatureRepository) Delete(ctx context.Context, tx *gorm.DB, id string, user auth.JWTPayload) error {
	sr.logger.Debugf("Delete signature with id: %d \n", id)

	db := sr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Signature{}).Where(model.Signature{
		BaseModel: model.BaseModel{
			ID: id,
		},
		UserID: user.ID,
	}).Delete(&model.Signature{}).Error; err != nil {
		sr.logger.Errorf("Failed to delete signature: %v", err)
		return err
	}

	return nil
}

func (sr SignatureRepository) GetById(ctx context.Context, tx *gorm.DB, id string, user auth.JWTPayload) (*model.Signature, error) {
	sr.logger.Debugf("Get signature with id: %d \n", id)

	db := sr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	sig := model.Signature{}

	if err := db.WithContext(ctx).Model(&model.Signature{}).Preload("SignatureFile").Where(model.Signature{
		BaseModel: model.BaseModel{
			ID: id,
		},
		UserID: user.ID,
	}).First(&sig).Error; err != nil {
		sr.logger.Errorf("Failed to get signature by id: %v", err)
		return nil, err
	}

	return &sig, nil
}
