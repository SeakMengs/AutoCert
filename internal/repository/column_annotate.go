package repository

import (
	"context"
	"errors"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type ColumnAnnotateRepository struct {
	*baseRepository
}

func (car ColumnAnnotateRepository) Create(ctx context.Context, tx *gorm.DB, ca *model.ColumnAnnotate) error {
	car.logger.Debugf("Create column annotate with data: %v \n", ca)

	db := car.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Create(&ca).Error; err != nil {
		car.logger.Errorf("Failed to create column annotate: %v", err)
		return err
	}

	return nil
}

func (car ColumnAnnotateRepository) Update(ctx context.Context, tx *gorm.DB, ca *model.ColumnAnnotate) error {
	car.logger.Debugf("Update column annotate with data: %v \n", ca)

	db := car.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if ca.ID == "" {
		car.logger.Errorf("Failed to update column annotate: ID is empty")
		return errors.New("ID cannot be empty for update operation")
	}

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: ca.ID,
		},
	}).Updates(&ca).Error; err != nil {
		car.logger.Errorf("Failed to update column annotate: %v", err)
		return err
	}

	return nil
}

func (car ColumnAnnotateRepository) UpdateTextFitRectBox(ctx context.Context, tx *gorm.DB, caId string, TextFitRectBox bool) error {
	car.logger.Debugf("Update text fit rect box of column annotate with id: %s \n", caId)

	db := car.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Select("text_fit_rect_box").Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: caId,
		},
	}).Updates(model.ColumnAnnotate{
		TextFitRectBox: TextFitRectBox,
	}).Error; err != nil {
		car.logger.Errorf("Failed to update text fit rect box of column annotate: %v", err)
		return err
	}

	return nil
}

func (car ColumnAnnotateRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	car.logger.Debugf("Delete column annotate with id: %d \n", id)

	db := car.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: id,
		},
	}).Delete(&model.ColumnAnnotate{}).Error; err != nil {
		car.logger.Errorf("Failed to delete column annotate: %v", err)
		return err
	}

	return nil
}
