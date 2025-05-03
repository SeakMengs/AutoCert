package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type CertificateRepository struct {
	*baseRepository
}

func (cr CertificateRepository) Create(ctx context.Context, tx *gorm.DB, ca *model.Certificate) (*model.Certificate, error) {
	cr.logger.Debugf("Create certificate: %s", ca.Number)

	db := cr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Certificate{}).Create(ca).Error; err != nil {
		return ca, err
	}

	return ca, nil
}

func (plr CertificateRepository) GetByProjectId(ctx context.Context, tx *gorm.DB, projectId string) ([]model.Certificate, error) {
	plr.logger.Debugf("Get certificates by project id: %s", projectId)

	db := plr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var certificates []model.Certificate
	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(model.Certificate{
		ProjectID: projectId,
	}).Preload("CertificateFile").Find(&certificates).Error; err != nil {
		return certificates, err
	}

	return certificates, nil
}
