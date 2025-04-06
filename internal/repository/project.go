package repository

import (
	"context"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type ProjectRepository struct {
	*baseRepository
}

func (pr ProjectRepository) Create(ctx context.Context, tx *gorm.DB, project *model.Project) (*model.Project, error) {
	pr.logger.Debugf("Create project with data: %v \n", project)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.Project{}).Create(project).Error; err != nil {
		return project, err
	}

	return project, nil
}
