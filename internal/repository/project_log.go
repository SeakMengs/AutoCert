package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type ProjectLogRepository struct {
	*baseRepository
}

func (plr ProjectLogRepository) GetByProjectId(ctx context.Context, tx *gorm.DB, projectId string) ([]*model.ProjectLog, error) {
	plr.logger.Debugf("Get project logs by project id: %s", projectId)

	db := plr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var logs []*model.ProjectLog
	if err := db.WithContext(ctx).Model(&model.ProjectLog{}).Where(model.ProjectLog{
		ProjectID: projectId,
	}).Order("timestamp asc").Find(&logs).Error; err != nil {
		return logs, err
	}

	return logs, nil
}
