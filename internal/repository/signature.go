package repository

import (
	"context"

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
