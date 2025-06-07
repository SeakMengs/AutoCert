package controller

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type SignatureController struct {
	*baseController
}

var ALLOWED_SIGNATURE_FILE_TYPE = []string{".png", ".svg"}

const (
	ErrSignatureIdRequired = "signature id is required"
)

func getSignatureDirectoryPath(userId string) string {
	return fmt.Sprintf("users/%s/signatures", userId)
}

func toSignatureDirectoryPath(userId string, filename string) string {
	return filepath.Join(getSignatureDirectoryPath(userId), filepath.Base(filename))
}

// TODO: store pub key
// TODO: limit file size
func (sc SignatureController) AddSignature(ctx *gin.Context) {
	// type Request struct {
	// 	// Title string `json:"title" form:"title" binding:"required,strNotEmpty,min=1,max=100"`
	// 	// Page  int    `json:"page" form:"page" binding:"required,number,gte=1"`
	// }
	// var body Request

	user, err := sc.getAuthUser(ctx)
	if err != nil {
		sc.app.Logger.Errorf("Failed to get auth user: %v", err)
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	// err = ctx.ShouldBind(&body)
	// if err != nil {
	// 	sc.app.Logger.Errorf("Failed to bind request: %v", err)
	// 	util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err), nil)
	// 	return
	// }

	sigFile, err := ctx.FormFile("signatureFile")
	if err != nil {
		sc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "No signature file uploaded", util.GenerateErrorMessages(errors.New("signature file is required"), "signatureFile"), nil)
		return
	}

	ext := filepath.Ext(sigFile.Filename)
	if !slices.Contains(ALLOWED_SIGNATURE_FILE_TYPE, ext) {
		sc.app.Logger.Errorf("Failed to add signature: invalid file type %s", ext)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid file type", util.GenerateErrorMessages(errors.New("invalid file type"), "signatureFile"), nil)
		return
	}

	info, err := util.UploadFileToS3ByFileHeader(sigFile, &util.FileUploadOptions{
		DirectoryPath: getSignatureDirectoryPath(user.ID),
		UniquePrefix:  true,
		Bucket:        sc.app.Config.Minio.BUCKET,
		S3:            sc.app.S3,
	})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to upload file", util.GenerateErrorMessages(err), nil)
		return
	}

	tx := sc.app.Repository.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to add signature", util.GenerateErrorMessages(errors.New("failed to add signature")), nil)
			return
		}
	}()

	sig := model.Signature{
		UserID: user.ID,
		SignatureFile: model.File{
			FileName:       toSignatureDirectoryPath(user.ID, sigFile.Filename),
			UniqueFileName: info.Key,
			BucketName:     info.Bucket,
			Size:           info.Size,
		},
	}

	_, err = sc.app.Repository.Signature.Create(ctx, nil, &sig)
	if err != nil {
		// delete the file from s3 if signature creation failed
		if err := sc.app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
			sc.app.Logger.Error(err)
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to delete signature file", util.GenerateErrorMessages(err), nil)
			return
		}

		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create signature", util.GenerateErrorMessages(err), nil)
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create signature", util.GenerateErrorMessages(err), nil)
		return
	}

	var sigUrl string
	if sig.SignatureFileID != "" {
		sigUrl, err = sig.SignatureFile.ToPresignedUrl(ctx, sc.app.S3)
		if err != nil {
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get signature file URL", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	util.ResponseSuccess(ctx, gin.H{
		"signature": gin.H{
			"id":  sig.ID,
			"url": sigUrl,
		},
	})
}

func (sc SignatureController) RemoveSignature(ctx *gin.Context) {
	signatureId := ctx.Params.ByName("signatureId")
	if signatureId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Signature id is required", util.GenerateErrorMessages(errors.New(ErrSignatureIdRequired), "signatureId"), nil)
		return
	}

	user, err := sc.getAuthUser(ctx)
	if err != nil {
		sc.app.Logger.Errorf("Failed to get auth user: %v", err)
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	tx := sc.app.Repository.DB.Begin()
	defer tx.Commit()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to remove signature", util.GenerateErrorMessages(errors.New("failed to remove signature")), nil)
			return
		}
	}()

	sig, err := sc.app.Repository.Signature.GetById(ctx, tx, signatureId, *user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to remove signature", util.GenerateErrorMessages(errors.New("signature not found"), "signatureId"), nil)
			return
		}

		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to remove signature", util.GenerateErrorMessages(err), nil)
		return
	}

	err = sc.app.Repository.Signature.Delete(ctx, tx, signatureId, *user)
	if err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to remove signature", util.GenerateErrorMessages(err), nil)
		return
	}

	if sig.SignatureFileID != "" {
		err := sig.SignatureFile.Delete(ctx, sc.app.S3)
		if err != nil {
			// Intentionally not return failed because even if delete file fail, it doesn't affect the system.
			sc.app.Logger.Errorf("failed to delete signature file from storage with err: %v", err)
		}
	}

	util.ResponseSuccess(ctx, nil)
}

func (sc SignatureController) GetSignatureById(ctx *gin.Context) {
	signatureId := ctx.Params.ByName("signatureId")
	if signatureId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Signature id is required", util.GenerateErrorMessages(errors.New(ErrSignatureIdRequired), "signatureId"), nil)
		return
	}

	user, err := sc.getAuthUser(ctx)
	if err != nil {
		sc.app.Logger.Errorf("Failed to get auth user: %v", err)
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	sig, err := sc.app.Repository.Signature.GetById(ctx, nil, signatureId, *user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to get signature", util.GenerateErrorMessages(errors.New("signature not found"), "signatureId"), nil)
			return
		}

		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get signature", util.GenerateErrorMessages(err), nil)
		return
	}

	var sigUrl string
	if sig.SignatureFileID != "" {
		sigUrl, err = sig.SignatureFile.ToPresignedUrl(ctx, sc.app.S3)
		if err != nil {
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get signature file URL", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	util.ResponseSuccess(ctx, gin.H{
		"signature": gin.H{
			"url":      sigUrl,
			"filename": sig.SignatureFile.ToBaseFilename(),
		},
	})
}
