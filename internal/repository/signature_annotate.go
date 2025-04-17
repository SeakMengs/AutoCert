package repository

import (
	"context"
	"errors"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type SignatureAnnotateRepository struct {
	*baseRepository
}

func (sar SignatureAnnotateRepository) Create(ctx context.Context, tx *gorm.DB, sa *model.SignatureAnnotate) error {
	sar.logger.Debugf("Create or update signature annotate with data: %v \n", sa)

	db := sar.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Create(&sa).Error; err != nil {
		sar.logger.Errorf("Failed to create signature annotate: %v", err)
		return err
	}

	return nil
}

func (sar SignatureAnnotateRepository) Update(ctx context.Context, tx *gorm.DB, sa map[string]any) error {
	sar.logger.Debugf("Update signature annotate with data: %v \n", sa)

	db := sar.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if sa["id"] == "" {
		sar.logger.Errorf("Failed to update signature annotate: ID is empty")
		return errors.New("ID cannot be empty for update operation")
	}

	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Where(model.SignatureAnnotate{
		BaseModel: model.BaseModel{
			ID: sa["id"].(string),
		},
	}).Updates(&sa).Error; err != nil {
		sar.logger.Errorf("Failed to update signature annotate: %v", err)
		return err
	}

	return nil
}

func (sar SignatureAnnotateRepository) InviteSignatory(ctx context.Context, tx *gorm.DB, caId string) error {
	sar.logger.Debugf("Invite signatory to signature annotate with id: %s \n", caId)

	db := sar.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Select("status").Where(model.SignatureAnnotate{
		BaseModel: model.BaseModel{
			ID: caId,
		},
		Status: constant.SignatoryStatusNotInvited,
	}).Updates(&model.SignatureAnnotate{
		Status: constant.SignatoryStatusInvited,
	}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("signatory not found or already invited")
		}

		sar.logger.Errorf("Failed to invite signatory: %v", err)
		return err
	}

	return nil
}

func (sar SignatureAnnotateRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	sar.logger.Debugf("Delete signature annotate with id: %d \n", id)

	db := sar.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Where(model.SignatureAnnotate{
		BaseModel: model.BaseModel{
			ID: id,
		},
	}).Delete(&model.SignatureAnnotate{}).Error; err != nil {
		sar.logger.Errorf("Failed to delete signature annotate: %v", err)
		return err
	}

	return nil
}
