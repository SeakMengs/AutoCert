package controller

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
		ID                   string                        `json:"id"`
		Title                string                        `json:"title"`
		IsPublic             bool                          `json:"isPublic"`
		Status               constant.ProjectStatus        `json:"status"`
		CreatedAt            *time.Time                    `json:"createdAt"`
		Signatories          []repository.ProjectSignatory `json:"signatories"`
		Logs                 []ProjectLog                  `json:"logs"`
		Certificates         []Certificate                 `json:"certificates"`
		CertificateMergedUrl string                        `json:"certificateMergedUrl,omitempty"`
		CertificateZipUrl    string                        `json:"certificateZipUrl,omitempty"`
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
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), "notFound"), nil)
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

	signatories, err := cc.app.Repository.Project.GetProjectSignatories(ctx, nil, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project signatories", util.GenerateErrorMessages(err), nil)
		return
	}

	if len(*certificates) == 0 {
		certificates = &[]model.Certificate{}
	}

	if len(logs) == 0 {
		logs = []*model.ProjectLog{}
	}

	if len(signatories) == 0 {
		signatories = []repository.ProjectSignatory{}
	}

	var certMergedUrl, certZipUrl string
	certificateList := make([]Certificate, 0, len(*certificates))

	for _, ca := range *certificates {
		switch ca.Type {
		case autocert.CertificateTypeNormal:
			cert := Certificate{
				ID:        ca.ID,
				Number:    ca.Number,
				CreatedAt: ca.CreatedAt.String(),
			}
			if ca.CertificateFileId != "" {
				url, err := ca.CertificateFile.ToPresignedUrl(ctx, cc.app.S3)
				if err != nil {
					util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get presigned URL for certificate", util.GenerateErrorMessages(err), nil)
					return
				}
				cert.CertificateUrl = url
			}
			certificateList = append(certificateList, cert)
		case autocert.CertificateTypeMerged:
			if ca.CertificateFileId != "" {
				url, err := ca.CertificateFile.ToPresignedUrl(ctx, cc.app.S3)
				if err != nil {
					util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get presigned URL for merged certificate", util.GenerateErrorMessages(err), nil)
					return
				}
				certMergedUrl = url
			}
		case autocert.CertificateTypeZip:
			if ca.CertificateFileId != "" {
				url, err := ca.CertificateFile.ToPresignedUrl(ctx, cc.app.S3)
				if err != nil {
					util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get presigned URL for zip certificate", util.GenerateErrorMessages(err), nil)
					return
				}
				certZipUrl = url
			}
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
			ID:                   project.ID,
			Title:                project.Title,
			Status:               project.Status,
			IsPublic:             project.IsPublic,
			CreatedAt:            project.CreatedAt,
			Certificates:         certificateList,
			Logs:                 logList,
			Signatories:          signatories,
			CertificateMergedUrl: certMergedUrl,
			CertificateZipUrl:    certZipUrl,
		},
	})
}

// TODO: remove
func (cc CertificateController) CertificatesToZipByProjectId(ctx *gin.Context) {
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
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), "notFound"), nil)
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

	if len(*certificates) == 0 {
		util.ResponseFailed(ctx, http.StatusBadRequest, "No certificates found for this project", util.GenerateErrorMessages(errors.New("no certificate in this project"), "certificates"), nil)
		return
	}

	tempDir, err := util.MkdirTemp("certificates_zip_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempDir)

	for _, certificate := range *certificates {
		if certificate.CertificateFileId != "" {
			base := filepath.Base(certificate.CertificateFile.FileName)
			filePath := filepath.Join(tempDir, base)

			err := certificate.CertificateFile.DownloadToLocal(ctx, cc.app.S3, filePath)
			if err != nil {
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to download certificate file", util.GenerateErrorMessages(err), nil)
				return
			}
		}
	}

	zipFilePath, err := util.CreateTemp("autocert_certificates_*.zip")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create zip file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(zipFilePath.Name())
	defer zipFilePath.Close()

	err = autocert.ZipDir(tempDir, zipFilePath.Name())
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create zip file", util.GenerateErrorMessages(err), nil)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=certificates.zip")
	ctx.File(zipFilePath.Name())
}

// TODO: remove
func (cc CertificateController) MergeCertificatesByProjectId(ctx *gin.Context) {
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
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), "notFound"), nil)
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

	if len(*certificates) == 0 {
		util.ResponseFailed(ctx, http.StatusBadRequest, "No certificates found for this project", util.GenerateErrorMessages(errors.New("no certificate in this project"), "certificates"), nil)
		return
	}

	tempDir, err := util.MkdirTemp("certificates_zip_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempDir)

	inFile := make([]string, 0, len(*certificates))
	for _, certificate := range *certificates {
		if certificate.CertificateFileId != "" {
			base := filepath.Base(certificate.CertificateFile.FileName)
			filePath := filepath.Join(tempDir, base)

			err := certificate.CertificateFile.DownloadToLocal(ctx, cc.app.S3, filePath)
			if err != nil {
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to download certificate file", util.GenerateErrorMessages(err), nil)
				return
			}

			inFile = append(inFile, filePath)
		}
	}

	mergeOutPut, err := util.CreateTemp("autocert_merged_certificates_*.pdf")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create merged PDF file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(mergeOutPut.Name())

	err = autocert.MergePdf(inFile, mergeOutPut.Name())
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to merge PDF files", util.GenerateErrorMessages(err), nil)
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=merged_certificates.pdf")
	ctx.File(mergeOutPut.Name())
}

func (cc CertificateController) GetGeneratedCertificateById(ctx *gin.Context) {
	certificateId := ctx.Param("certificateId")
	if certificateId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Certificate id is required", util.GenerateErrorMessages(errors.New("certificateId is required"), "certificateId"), nil)
		return
	}

	certificate, err := cc.app.Repository.Certificate.GetById(ctx, nil, certificateId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.ResponseFailed(ctx, http.StatusNotFound, "Certificate not found", util.GenerateErrorMessages(errors.New("certificate not found"), "notFound"), nil)
			return
		}

		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get certificate", util.GenerateErrorMessages(err), nil)
		return
	}

	if certificate == nil || certificate.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Certificate not found", util.GenerateErrorMessages(errors.New("certificate not found"), "notFound"), nil)
		return
	}

	if !certificate.Project.IsPublic {
		util.ResponseFailed(ctx, http.StatusForbidden, "This project of requested certificate is not public", util.GenerateErrorMessages(errors.New("the project of requested certificate is not public"), "forbidden"), nil)
		return
	}

	if certificate.CertificateFileId == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Certificate file not found", util.GenerateErrorMessages(errors.New("certificate file not found"), "notFound"), nil)
		return
	}

	url, err := certificate.CertificateFile.ToPresignedUrl(ctx, cc.app.S3)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get presigned URL for certificate", util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"id":             certificate.ID,
		"certificateUrl": url,
		"number":         certificate.Number,
		"issuer":         certificate.Project.User.FirstName + " " + certificate.Project.User.LastName,
		"issuedAt":       certificate.CreatedAt.String(),
		"projectTitle":   certificate.Project.Title,
	})
}
