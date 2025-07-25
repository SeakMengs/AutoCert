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

func (car ColumnAnnotateRepository) Update(ctx context.Context, tx *gorm.DB, ca map[string]any) error {
	car.logger.Debugf("Update column annotate with data: %v \n", ca)

	db := car.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if ca["id"] == "" {
		car.logger.Errorf("Failed to update column annotate: ID is empty")
		return errors.New("ID cannot be empty for update operation")
	}

	// remove key that cannot be updated
	var forbiddenKeys = []string{"created_at", "updated_at"}

	for _, key := range forbiddenKeys {
		delete(ca, key)
	}

	if err := db.WithContext(ctx).Model(&model.ColumnAnnotate{}).Where(model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: ca["id"].(string),
		},
	}).Updates(&ca).Error; err != nil {
		car.logger.Errorf("Failed to update column annotate: %v", err)
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
