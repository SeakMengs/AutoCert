package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileController struct {
	*baseController
}

func CacheRequest(ctx *gin.Context, time time.Duration) {
	ctx.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(time.Seconds())))
}

func (fc FileController) ServeProjectThumbnail(ctx *gin.Context) {
	selectedPages := "1"
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	user, roles, project, err := fc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "project"), nil)
		return
	}

	if !util.HasRole(user.Email, roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		if restricted, domain := util.IsRestrictedByEmailDomain(user.Email, roles); restricted {
			util.ResponseRestrictDomain(ctx, domain)
			return
		}

		util.ResponseNoPermission(ctx)
		return
	}

	tempOutDir, err := util.MkdirTemp("autocert_pdf_thumbnail_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempOutDir)
	defer func() {
		if err := recover(); err != nil {
			os.RemoveAll(tempOutDir)
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Error processing request", util.GenerateErrorMessages(errors.New("internal server error")), nil)
			return
		}
	}()

	pdfPath := fmt.Sprintf("%s/%s.pdf", tempOutDir, projectId)

	err = project.TemplateFile.DownloadToLocal(ctx, fc.app.S3, pdfPath)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error downloading PDF file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(pdfPath)

	if _, err := os.Stat(pdfPath); errors.Is(err, os.ErrNotExist) {
		util.ResponseFailed(ctx, http.StatusNotFound, "PDF file not found", util.GenerateErrorMessages(err), nil)
		return
	}

	thumbnailPath, err := autocert.PdfToThumbnailByPage(pdfPath, tempOutDir, selectedPages, 512, 512, autocert.ThumbnailFormatWebP)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error converting PDF to PNG", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(thumbnailPath)

	CacheRequest(ctx, time.Minute*10)
	ctx.File(thumbnailPath)
}

// TODO: remove
func (fc FileController) ServeProjectCertificateNumberThumbnail(ctx *gin.Context) {
	selectedPages := "1"
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	certificateNumber := ctx.Params.ByName("certificateNumber")
	if certificateNumber == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Certificate number is required", util.GenerateErrorMessages(errors.New("certificate number is required"), "certificateNumber"), nil)
		return
	}

	user, roles, project, err := fc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), "project"), nil)
		return
	}

	if !util.HasRole(user.Email, roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		if restricted, domain := util.IsRestrictedByEmailDomain(user.Email, roles); restricted {
			util.ResponseRestrictDomain(ctx, domain)
			return
		}

		util.ResponseNoPermission(ctx)
		return
	}

	certNumber, err := strconv.Atoi(certificateNumber)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid certificate number", util.GenerateErrorMessages(err, "certificateNumber"), nil)
		return
	}

	cert, err := fc.app.Repository.Certificate.GetByNumber(ctx, nil, projectId, certNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.ResponseFailed(ctx, http.StatusNotFound, "Certificate not found", util.GenerateErrorMessages(errors.New("certificate not found"), "certificateNumber"), nil)
			return
		}

		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get certificate", util.GenerateErrorMessages(err), nil)
		return
	}

	if cert == nil || cert.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Certificate not found", util.GenerateErrorMessages(errors.New("certificate not found"), "certificate"), nil)
		return
	}

	tempOutDir, err := util.MkdirTemp("autocert_cert_pdf_thumbnail_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempOutDir)
	defer func() {
		if err := recover(); err != nil {
			os.RemoveAll(tempOutDir)

			util.ResponseFailed(ctx, http.StatusInternalServerError, "Error processing request", util.GenerateErrorMessages(errors.New("internal server error")), nil)
			return
		}
	}()

	pdfPath := fmt.Sprintf("%s/%s_%s.pdf", tempOutDir, projectId, certificateNumber)

	err = cert.CertificateFile.DownloadToLocal(ctx, fc.app.S3, pdfPath)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error downloading PDF file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(pdfPath)

	if _, err := os.Stat(pdfPath); errors.Is(err, os.ErrNotExist) {
		util.ResponseFailed(ctx, http.StatusNotFound, "PDF file not found", util.GenerateErrorMessages(err), nil)
		return
	}

	thumbnailPath, err := autocert.PdfToThumbnailByPage(pdfPath, tempOutDir, selectedPages, 512, 512, autocert.ThumbnailFormatWebP)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error converting PDF to PNG", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(thumbnailPath)

	CacheRequest(ctx, time.Minute*10)
	ctx.File(thumbnailPath)
}
