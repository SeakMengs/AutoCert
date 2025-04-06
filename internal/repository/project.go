package repository

import (
	"context"
	"time"

	"github.com/SeakMengs/AutoCert/internal/auth"
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

func (pr ProjectRepository) GetRoleOfProject(ctx context.Context, tx *gorm.DB, projectID string, authUser *auth.JWTPayload) (constant.ProjectRole, *model.Project, error) {
	pr.logger.Debugf("Get role of project with projectID: %s and userID: %s \n", projectID, authUser.ID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var project model.Project
	var role constant.ProjectRole
	if err := db.WithContext(ctx).Model(&model.Project{}).Where(&model.Project{
		BaseModel: model.BaseModel{
			ID: projectID,
		},
		UserID: authUser.ID,
	}).Preload("TemplateFile").First(&project).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return constant.ProjectRoleNone, nil, err
		}
	}
	if project.ID != "" {
		role = constant.ProjectRoleOwner
		return role, &project, nil
	}

	var signature model.SignatureAnnotate
	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Where(&model.SignatureAnnotate{
		Email: authUser.Email,
		BaseAnnotateModel: model.BaseAnnotateModel{
			ProjectID: project.ID,
		},
	}).First(&signature).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return constant.ProjectRoleNone, nil, err
		}
	}

	if signature.ID != "" {
		role = constant.ProjectRoleSignatory
	} else {
		role = constant.ProjectRoleNone
	}

	return role, &project, nil
}

type ProjectSignatory struct {
	Email      string                   `json:"email"`
	ProfileUrl string                   `json:"profileUrl"`
	Status     constant.SignatoryStatus `json:"status"`
}

func (pr ProjectRepository) GetProjectSignatories(ctx context.Context, tx *gorm.DB, projectID string) ([]ProjectSignatory, error) {
	pr.logger.Debugf("Get project signatories information with projectID: %s \n", projectID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var signatories []ProjectSignatory
	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).
		Select("users.email, users.profile_url, signature_annotates.status").
		Joins("JOIN users ON signature_annotates.email = users.email").
		Where("signature_annotates.project_id = ?", projectID).
		Scan(&signatories).Error; err != nil {
		return nil, err
	}

	return signatories, nil
}

type ProjectResponse struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	TemplateUrl string                 `json:"templateUrl"`
	IsPublic    bool                   `json:"isPublic"`
	Signatories []ProjectSignatory     `json:"signatories"`
	Status      constant.ProjectStatus `json:"status"`
	CreatedAt   *time.Time             `json:"createdAt"`
}

func (pr ProjectRepository) GetProjectsForOwner(ctx context.Context, tx *gorm.DB, authUser *auth.JWTPayload, search string, status []constant.ProjectStatus, page, pageSize uint) ([]ProjectResponse, int64, error) {
	pr.logger.Debugf("Get projects for owner with userID: %s \n", authUser.ID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var projects []model.Project
	query := db.WithContext(ctx).Model(&model.Project{}).Preload("TemplateFile").
		Where("user_id = ?", authUser.ID)

	if len(status) > 0 {
		query = query.Where("projects.status IN (?)", status)
	}

	if search != "" {
		query = query.Where("projects.title ILIKE ?", "%"+search+"%")
	}

	if err := query.Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	var totalProjects int64
	if err := db.WithContext(ctx).Model(&model.Project{}).
		Where("user_id = ?", authUser.ID).
		Count(&totalProjects).Error; err != nil {
		return nil, 0, err
	}

	var projectRes []ProjectResponse

	for _, project := range projects {
		signatories, err := pr.GetProjectSignatories(ctx, nil, project.ID)
		if err != nil {
			return nil, 0, err
		}
		if len(signatories) == 0 {
			signatories = []ProjectSignatory{}
		}

		templateUrl, err := project.TemplateFile.ToPresignedUrl(ctx, pr.s3)
		if err != nil {
			return nil, 0, err
		}
		projectRes = append(projectRes, ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			TemplateUrl: templateUrl,
			IsPublic:    project.IsPublic,
			Signatories: signatories,
			Status:      project.Status,
			CreatedAt:   project.CreatedAt,
		})
	}

	return projectRes, totalProjects, nil
}

func (pr ProjectRepository) GetProjectsForSignatory(ctx context.Context, tx *gorm.DB, authUser *auth.JWTPayload, search string, status []constant.ProjectStatus, page, pageSize uint) ([]ProjectResponse, int64, error) {
	pr.logger.Debugf("Get projects for signatory with userID: %s \n", authUser.ID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var projects []model.Project
	query := db.WithContext(ctx).Model(&model.Project{}).Preload("TemplateFile").
		Joins("JOIN signature_annotates ON projects.id = signature_annotates.project_id").
		Where("signature_annotates.email = ?", authUser.Email)

	if len(status) > 0 {
		query = query.Where("projects.status IN (?)", status)
	}

	if search != "" {
		query = query.Where("projects.title ILIKE ?", "%"+search+"%")
	}

	if err := query.Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	var totalProjects int64
	if err := db.WithContext(ctx).Model(&model.Project{}).
		Joins("JOIN signature_annotates ON projects.id = signature_annotates.project_id").
		Where("signature_annotates.email = ?", authUser.Email).
		Count(&totalProjects).Error; err != nil {
		return nil, 0, err
	}

	var projectRes []ProjectResponse

	for _, project := range projects {
		signatories, err := pr.GetProjectSignatories(ctx, nil, project.ID)
		if err != nil {
			return nil, 0, err
		}
		if len(signatories) == 0 {
			signatories = []ProjectSignatory{}
		}

		templateUrl, err := project.TemplateFile.ToPresignedUrl(ctx, pr.s3)
		if err != nil {
			return nil, 0, err
		}
		projectRes = append(projectRes, ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			TemplateUrl: templateUrl,
			IsPublic:    project.IsPublic,
			Status:      project.Status,
			Signatories: signatories,
			CreatedAt:   project.CreatedAt,
		})
	}

	return projectRes, totalProjects, nil
}
