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

	if err := db.WithContext(ctx).Model(&model.Project{}).Create(&project).Error; err != nil {
		return project, err
	}

	return project, nil
}

func (pr ProjectRepository) GetRoleOfProject(ctx context.Context, tx *gorm.DB, projectID string, authUser *auth.JWTPayload) ([]constant.ProjectRole, *model.Project, error) {
	pr.logger.Debugf("Get role of project with projectID: %s and userID: %s \n", projectID, authUser.ID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var project model.Project
	var role []constant.ProjectRole
	if err := db.WithContext(ctx).Model(&model.Project{}).Where(&model.Project{
		BaseModel: model.BaseModel{
			ID: projectID,
		},
		UserID: authUser.ID,
	}).Preload("TemplateFile").Preload("CSVFile").Preload("SignatureAnnotates.SignatureFile").Preload("ColumnAnnotates").First(&project).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, nil, err
		}
	}

	if project.ID != "" {
		role = append(role, constant.ProjectRoleOwner)
	}

	var signature model.SignatureAnnotate
	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).Where(&model.SignatureAnnotate{
		Email: authUser.Email,
		BaseAnnotateModel: model.BaseAnnotateModel{
			ProjectID: project.ID,
		},
	}).Where("signature_annotates.status IN (?)", []constant.SignatoryStatus{constant.SignatoryStatusInvited, constant.SignatoryStatusSigned}).First(&signature).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, nil, err
		}
	}

	if signature.ID != "" {
		role = append(role, constant.ProjectRoleSignatory)
	}

	return role, &project, nil
}

type ProjectSignatory struct {
	Email      string                   `json:"email"`
	ProfileUrl string                   `json:"profileUrl"`
	Status     constant.SignatoryStatus `json:"status"`
}

// Doesn't care about the status of the signatory
func (pr ProjectRepository) GetProjectSignatories(ctx context.Context, tx *gorm.DB, projectID string) ([]ProjectSignatory, error) {
	pr.logger.Debugf("Get project signatories information with projectID: %s \n", projectID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var signatories []ProjectSignatory
	if err := db.WithContext(ctx).Model(&model.SignatureAnnotate{}).
		Select("users.email, users.profile_url, signature_annotates.status").
		Joins("JOIN users ON signature_annotates.email = users.email").
		Where(model.SignatureAnnotate{
			BaseAnnotateModel: model.BaseAnnotateModel{
				ProjectID: projectID,
			},
		}).
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
	Status      constant.ProjectStatus `json:"status"`
	CreatedAt   *time.Time             `json:"createdAt"`
	Signatories []ProjectSignatory     `json:"signatories"`
}

func (pr ProjectRepository) GetProjectsForOwner(ctx context.Context, tx *gorm.DB, authUser *auth.JWTPayload, search string, status []constant.ProjectStatus, page, pageSize uint) ([]ProjectResponse, int64, error) {
	pr.logger.Debugf("Get projects for owner with userID: %s \n", authUser.ID)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var projects []model.Project
	query := db.WithContext(ctx).Model(&model.Project{}).Preload("TemplateFile").
		Where(model.Project{
			UserID: authUser.ID,
		})

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
		Where(model.Project{
			UserID: authUser.ID,
		}).
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
		Where("signature_annotates.email = ?", authUser.Email).
		Where("signature_annotates.status IN (?)", []constant.SignatoryStatus{constant.SignatoryStatusInvited, constant.SignatoryStatusSigned}).
		Group("projects.id")

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
		Where("signature_annotates.status IN (?)", []constant.SignatoryStatus{constant.SignatoryStatusInvited, constant.SignatoryStatusSigned}).
		Group("projects.id").
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

func (pr ProjectRepository) UpdateSetting(ctx context.Context, tx *gorm.DB, projectId string, embedQr bool) error {
	pr.logger.Debugf("Update project setting with projectId: %s and embedQr: %v \n", projectId, embedQr)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	// Need to select because gorm does not allow none-zero value to be updated unless selected
	if err := db.WithContext(ctx).Model(&model.Project{}).Select("embed_qr").Where(&model.Project{
		BaseModel: model.BaseModel{
			ID: projectId,
		},
	}).Updates(&model.Project{
		EmbedQr: embedQr,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (pr ProjectRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, projectId string, status constant.ProjectStatus) error {
	pr.logger.Debugf("Update project status with projectId: %s and status: %v \n", projectId, status)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	// Need to select because gorm does not allow none-zero value to be updated unless selected
	if err := db.WithContext(ctx).Model(&model.Project{}).Select("status").Where(&model.Project{
		BaseModel: model.BaseModel{
			ID: projectId,
		},
	}).Updates(&model.Project{
		Status: status,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (pr ProjectRepository) UpdateCSVFile(ctx context.Context, tx *gorm.DB, project model.Project, csvFile *model.File) error {
	pr.logger.Debugf("Update project csv file with data: %v \n", project)

	db := pr.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if project.CSVFileID == "" {
		if err := db.WithContext(ctx).Model(&model.File{}).Create(csvFile).Error; err != nil {
			return err
		}
	} else {
		if err := db.WithContext(ctx).Model(&model.File{}).Where(&model.File{
			BaseModel: model.BaseModel{
				ID: project.CSVFileID,
			},
		}).Updates(csvFile).Error; err != nil {
			return err
		}
	}

	if err := db.WithContext(ctx).Model(&model.Project{}).Where(&model.Project{
		BaseModel: model.BaseModel{
			ID: project.ID,
		},
	}).Updates(&model.Project{
		CSVFileID: csvFile.ID,
	}).Error; err != nil {
		return err
	}

	return nil
}
