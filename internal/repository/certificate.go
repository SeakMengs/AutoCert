package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type CertificateRepository struct {
	*baseRepository
}

func (cr CertificateRepository) Create(ctx context.Context, tx *gorm.DB, ca *model.Certificate) (*model.Certificate, error) {
	// cr.logger.Debugf("Create certificate: %d", ca.Number)

	db := cr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Certificate{}).Create(ca).Error; err != nil {
		return ca, err
	}

	return ca, nil
}

func (cr CertificateRepository) CreateMany(ctx context.Context, tx *gorm.DB, certificates []*model.Certificate) ([]*model.Certificate, error) {
	cr.logger.Debugf("Create multiple certificates: %d", len(certificates))

	db := cr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	silentDB := db.Session(&gorm.Session{
		Logger: db.Logger.LogMode(logger.Silent),
	})

	if err := silentDB.WithContext(ctx).Model(&model.Certificate{}).Create(certificates).Error; err != nil {
		return certificates, err
	}

	return certificates, nil
}

// Return certificates, merged certificate, zip certificate, and total count
func (plr CertificateRepository) GetByProjectId(ctx context.Context, tx *gorm.DB, projectId string, page, pageSize uint) (*[]model.Certificate, *model.Certificate, *model.Certificate, int64, error) {
	plr.logger.Debugf("Get certificates by project id: %s", projectId)

	db := plr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var certificates []model.Certificate
	var certificateMerged model.Certificate
	var certificateZip model.Certificate
	total := int64(0)

	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(map[string]any{
		"project_id": projectId,
		"type":       autocert.CertificateTypeNormal,
	}).Count(&total).Error; err != nil {
		return &certificates, &certificateMerged, &certificateZip, total, err
	}

	query := db.WithContext(ctx).Model(&model.Certificate{}).Where(map[string]any{
		"project_id": projectId,
		"type":       autocert.CertificateTypeNormal,
	}).Preload("CertificateFile").Order("number asc")

	if err := query.Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&certificates).Error; err != nil {
		return &certificates, &certificateMerged, &certificateZip, total, err
	}

	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(model.Certificate{
		ProjectID: projectId,
		Type:      autocert.CertificateTypeMerged,
	}).Preload("CertificateFile").First(&certificateMerged).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return &certificates, &certificateMerged, &certificateZip, total, err
		}
	}

	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(model.Certificate{
		ProjectID: projectId,
		Type:      autocert.CertificateTypeZip,
	}).Preload("CertificateFile").First(&certificateZip).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return &certificates, &certificateMerged, &certificateZip, total, err
		}
	}

	return &certificates, &certificateMerged, &certificateZip, total, nil
}

func (plr CertificateRepository) GetByNumber(ctx context.Context, tx *gorm.DB, projectId string, number int) (*model.Certificate, error) {
	plr.logger.Debugf("Get certificate by number: %d, project id: %s", number, projectId)

	db := plr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var certificate model.Certificate
	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(map[string]any{"number": number, "project_id": projectId}).Preload("CertificateFile").First(&certificate).Error; err != nil {
		return &certificate, err
	}

	return &certificate, nil
}

func (plr CertificateRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (*model.Certificate, error) {
	plr.logger.Debugf("Get certificate by id: %s", id)

	db := plr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var certificate model.Certificate
	if err := db.WithContext(ctx).Model(&model.Certificate{}).Where(model.Certificate{
		BaseModel: model.BaseModel{
			ID: id,
		},
	}).Preload("CertificateFile").Preload("Project.User").First(&certificate).Error; err != nil {
		return &certificate, err
	}

	return &certificate, nil
}
