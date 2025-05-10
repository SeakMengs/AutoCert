package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type FileController struct {
	*baseController
}

func createBucketIfNotExists(s3 *minio.Client, bucketName string) error {
	exists, err := s3.BucketExists(context.Background(), bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = s3.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (fc FileController) ServeProjectThumbnail(ctx *gin.Context) {
	selectedPages := "1"
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, project, err := fc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "project"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project")), nil)
		return
	}

	tempOutDir, err := os.MkdirTemp("", "autocert_pdf_thumbnail_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempOutDir)

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

	pngFile, err := autocert.PdfToPngByPage(pdfPath, tempOutDir, selectedPages)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error converting PDF to PNG", util.GenerateErrorMessages(err), nil)
		return
	}

	if pngFile == nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating PNG file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(*pngFile)

	ctx.File(*pngFile)
}

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

	roles, project, err := fc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), "project"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project"), nil), nil)
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

	tempOutDir, err := os.MkdirTemp("", "autocert_cert_pdf_thumbnail_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempOutDir)

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

	pngFile, err := autocert.PdfToPngByPage(pdfPath, tempOutDir, selectedPages)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error converting PDF to PNG", util.GenerateErrorMessages(err), nil)
		return
	}

	if pngFile == nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating PNG file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(*pngFile)

	ctx.File(*pngFile)
}
