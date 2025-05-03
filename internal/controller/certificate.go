package controller

import (
	"errors"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

type CertificateController struct {
	*baseController
}

func (cc CertificateController) GetCertificatesByProjectId(ctx *gin.Context) {
	type Certificate struct {
		ID             string `json:"id"`
		Number         int    `json:"number"`
		CertificateUrl string `json:"certificateUrl"`
		CreatedAt      string `json:"createdAt"`
	}

	type ProjectLog struct {
		ID          string `json:"id"`
		Role        string `json:"role"`
		Action      string `json:"action"`
		Description string `json:"description"`
		Timestamp   string `json:"timestamp"`
	}

	type Project struct {
		ID           string                 `json:"id"`
		Title        string                 `json:"title"`
		IsPublic     bool                   `json:"isPublic"`
		Status       constant.ProjectStatus `json:"status"`
		Certificates []Certificate          `json:"certificates"`
		Logs         []ProjectLog           `json:"logs"`
	}

	type GetCertificatesByProjectIdResponse struct {
		Roles   []constant.ProjectRole `json:"roles"`
		Project Project                `json:"project"`
	}

	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, project, err := cc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "notFound"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project"), "forbidden"), nil)
		return
	}

	certificates, err := cc.app.Repository.Certificate.GetByProjectId(ctx, nil, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get certificates", util.GenerateErrorMessages(err), nil)
		return
	}

	logs, err := cc.app.Repository.ProjectLog.GetByProjectId(ctx, nil, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project logs", util.GenerateErrorMessages(err), nil)
		return
	}

	certificateList := make([]Certificate, len(certificates))
	for i, ca := range certificates {
		certificateList[i] = Certificate{
			ID:             ca.ID,
			Number:         ca.Number,
			CertificateUrl: "",
			CreatedAt:      ca.CreatedAt.String(),
		}

		if ca.CertificateFileId != "" {
			url, err := ca.CertificateFile.ToPresignedUrl(ctx, cc.app.S3)
			if err != nil {
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get presigned URL for certificate", util.GenerateErrorMessages(err), nil)
				return
			}

			certificateList[i].CertificateUrl = url
		}
	}

	logList := make([]ProjectLog, len(logs))
	for i, l := range logs {
		logList[i] = ProjectLog{
			ID:          l.ID,
			Role:        l.Role,
			Action:      l.Action,
			Description: l.Description,
			Timestamp:   l.Timestamp,
		}
	}

	util.ResponseSuccess(ctx, GetCertificatesByProjectIdResponse{
		Roles: roles,
		Project: Project{
			ID:           project.ID,
			Title:        project.Title,
			Status:       project.Status,
			IsPublic:     project.IsPublic,
			Certificates: certificateList,
			Logs:         logList,
		},
	})
}
