package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type ColumnAnnotateRepository struct {
	*baseRepository
}

func (pbr ColumnAnnotateRepository) Create(ctx context.Context, tx *gorm.DB, ca model.ColumnAnnotate) error {
	pbr.logger.Debugf("Create or update column annotate with data: %v \n", ca)

	db := pbr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Create(&ca).Error; err != nil {
		pbr.logger.Errorf("Failed to create column annotate: %v", err)
		return err
	}

	return nil
}

func (pbr ColumnAnnotateRepository) Update(ctx context.Context, tx *gorm.DB, ca model.ColumnAnnotate) error {
	pbr.logger.Debugf("Update column annotate with data: %v \n", ca)

	db := pbr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Select("*").Omit("created_at", "updated_at").Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: ca.ID,
		},
	}).Updates(ca).Error; err != nil {
		pbr.logger.Errorf("Failed to update column annotate: %v", err)
		return err
	}

	return nil
}

func (pbr ColumnAnnotateRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	pbr.logger.Debugf("Delete column annotate with id: %d \n", id)

	db := pbr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: id,
		},
	}).Delete(&model.ColumnAnnotate{}).Error; err != nil {
		pbr.logger.Errorf("Failed to delete column annotate: %v", err)
		return err
	}

	return nil
}
