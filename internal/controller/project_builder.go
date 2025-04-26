package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type ProjectBuilderController struct {
	*baseController
}

// TODO: add validation for each field
// TODO: correct data type, for now we accept everything from the model.{x}Annotate
type AnnotateType string

const (
	AnnotateTypeColumn    AnnotateType = "column"
	AnnotateTypeSignature AnnotateType = "signature"
)

type ColumnAnnotateState struct {
	model.ColumnAnnotate
	Type AnnotateType `json:"type" binding:"required" form:"type"`
}

type SignatureAnnotateState struct {
	model.SignatureAnnotate
	Type AnnotateType `json:"type" binding:"required" form:"type"`
}

type AutoCertSettings struct {
	QrCodeEnabled bool `json:"qrCodeEnabled" binding:"required" form:"qrCodeEnabled"`
}

type AnnotateColumnAdd struct {
	ColumnAnnotateState
	Page int `json:"page" binding:"required" form:"page"`
}

type AnnotateColumnUpdate struct {
	ColumnAnnotateState
	Page int `json:"page" binding:"required" form:"page"`
}

type AnnotateColumnRemove struct {
	ID string `json:"id" binding:"required" form:"id"`
}

type AnnotateSignatureAdd struct {
	SignatureAnnotateState
	Page int `json:"page" binding:"required" form:"page"`
}

type AnnotateSignatureUpdate struct {
	SignatureAnnotateState
	Page int `json:"page" binding:"required" form:"page"`
}

type AnnotateSignatureRemove struct {
	ID string `json:"id" binding:"required" form:"id"`
}

type AnnotateSignatureInvite struct {
	ID string `json:"id" binding:"required" form:"id"`
}

type AnnotateSignatureApprove struct {
	ID string `json:"id" binding:"required" form:"id"`
}

type SettingsUpdate struct {
	QrCodeEnabled bool `json:"qrCodeEnabled" binding:"required" form:"qrCodeEnabled"`
}

type TableUpdate struct {
	CSVFile *multipart.FileHeader `form:"csvFile" binding:"required"`
}

// AutoCertChangeEvent is a generic wrapper that holds the event type and raw payload.
type AutoCertChangeEvent struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data json.RawMessage            `json:"data" binding:"required" form:"data"`
}

const (
	ErrFailedToUpdateProjectBuilder = "Failed to update project builder"
)

// Take list of event types and their corresponding payloads, if at least one of the update fail, will revert all changes and respond with error
// For detail document of this API, check docs/bruno
func (pbc ProjectBuilderController) ProjectBuilder(ctx *gin.Context) {
	projectId := ctx.Param("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("projectId is required"), "projectId"), nil)
		return
	}

	roles, project, err := pbc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil {
		util.ResponseFailed(ctx, http.StatusNotFound, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("project not found"), "project"), nil)
		return
	}

	if project.Status != constant.ProjectStatusDraft {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("project is not in draft status"), "project"), nil)
		return
	}

	// Get the events JSON from the form.
	eventsJSON := ctx.PostForm("events")
	if eventsJSON == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("events is required"), "events"), nil)
		return
	}

	// Unmarshal the events JSON into an array.
	var events []AutoCertChangeEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("failed to parse events"), "events"), nil)
		return
	}

	// sort events by making table update last since it involve updating file which is not transactional
	var tableUpdateEvents []AutoCertChangeEvent
	var otherEvents []AutoCertChangeEvent

	for _, event := range events {
		if event.Type == constant.TableUpdate {
			tableUpdateEvents = append(tableUpdateEvents, event)
		} else {
			otherEvents = append(otherEvents, event)
		}
	}

	events = append(otherEvents, tableUpdateEvents...)

	tx := pbc.app.Repository.DB.Begin()
	defer tx.Commit()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("failed to process events"), "events"), nil)
			return
		}
	}()

	handlers := pbc.getEventHandlers()

	for idx, event := range events {
		pbc.app.Logger.Debugf("Processing event #%d, type %s", idx+1, event.Type)
		handler, ok := handlers[event.Type]

		if !ok {
			pbc.app.Logger.Errorf("Unknown event type: %s", event.Type)
			continue
		}

		if err := handler(ctx, tx, roles, project, event.Data); err != nil {
			tx.Rollback()
			pbc.app.Logger.Errorf("Failed to handle event %s: %v", event.Type, err)
			util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(err, nil, "events"), nil)
			return
		}
	}

	// delete the existing csv file after transaction is committed
	if len(tableUpdateEvents) > 0 && project.CSVFile.UniqueFileName != "" {
		pbc.app.S3.RemoveObject(ctx, project.CSVFile.BucketName, project.CSVFile.UniqueFileName, minio.RemoveObjectOptions{})
	}

	util.ResponseSuccess(ctx, nil)
}

type EventHandlerType func(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error

func (pbc ProjectBuilderController) getEventHandlers() map[constant.ProjectPermission]EventHandlerType {
	return map[constant.ProjectPermission]EventHandlerType{
		constant.AnnotateColumnAdd:        pbc.handleAnnotateColumnAdd,
		constant.AnnotateColumnUpdate:     pbc.handleAnnotateColumnUpdate,
		constant.AnnotateColumnRemove:     pbc.handleAnnotateColumnRemove,
		constant.AnnotateSignatureAdd:     pbc.handleAnnotateSignatureAdd,
		constant.AnnotateSignatureUpdate:  pbc.handleAnnotateSignatureUpdate,
		constant.AnnotateSignatureRemove:  pbc.handleAnnotateSignatureRemove,
		constant.AnnotateSignatureInvite:  pbc.handleAnnotateSignatureInvite,
		constant.AnnotateSignatureApprove: pbc.handleAnnotateSignatureApprove,
		constant.SettingsUpdate:           pbc.handleSettingsUpdate,
		constant.TableUpdate:              pbc.handleTableUpdate,
	}
}

func (pbc ProjectBuilderController) handleAnnotateColumnAdd(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateColumnAdd
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnAdd")
	}
	pbc.app.Logger.Debugf("AnnotateColumnAdd: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnAdd}) {
		return errors.New("you do not have permission to add column annotate")
	}

	err := pbc.app.Repository.ColumnAnnotate.Create(ctx, tx, &model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: payload.ID,
		},
		BaseAnnotateModel: model.BaseAnnotateModel{
			Page:      uint(payload.Page),
			X:         payload.X,
			Y:         payload.Y,
			Width:     payload.Width,
			Height:    payload.Height,
			Color:     payload.Color,
			ProjectID: project.ID,
		},
		Value:          payload.Value,
		FontName:       payload.FontName,
		FontSize:       payload.FontSize,
		FontColor:      payload.FontColor,
		FontWeight:     payload.FontWeight,
		TextFitRectBox: payload.TextFitRectBox,
	})
	if err != nil {
		return errors.New("failed to add column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateColumnUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateColumnUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnUpdate")
	}
	pbc.app.Logger.Debugf("AnnotateColumnUpdate: %+v \n", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnUpdate}) {
		return errors.New("you do not have permission to update column annotate")
	}

	err := pbc.app.Repository.ColumnAnnotate.Update(ctx, tx, map[string]any{
		"id":                payload.ID,
		"page":              uint(payload.Page),
		"x":                 payload.X,
		"y":                 payload.Y,
		"width":             payload.Width,
		"height":            payload.Height,
		"color":             payload.Color,
		"project_id":        project.ID,
		"value":             payload.Value,
		"font_name":         payload.FontName,
		"font_size":         payload.FontSize,
		"font_color":        payload.FontColor,
		"font_weight":       payload.FontWeight,
		"text_fit_rect_box": payload.TextFitRectBox,
	})
	if err != nil {
		return errors.New("failed to update column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateColumnRemove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateColumnRemove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnRemove")
	}
	pbc.app.Logger.Debugf("AnnotateColumnRemove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnRemove}) {
		return errors.New("you do not have permission to remove column annotate")
	}

	err := pbc.app.Repository.ColumnAnnotate.Delete(ctx, tx, payload.ID)
	if err != nil {
		return errors.New("failed to remove column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureAdd(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureAdd
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureAdd")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureAdd: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureAdd}) {
		return errors.New("you do not have permission to add signature annotate")
	}

	err := pbc.app.Repository.SignatureAnnotate.Create(ctx, tx, &model.SignatureAnnotate{
		BaseModel: model.BaseModel{
			ID: payload.ID,
		},
		BaseAnnotateModel: model.BaseAnnotateModel{
			Page:      uint(payload.Page),
			X:         payload.X,
			Y:         payload.Y,
			Width:     payload.Width,
			Height:    payload.Height,
			Color:     payload.Color,
			ProjectID: project.ID,
		},
		Status: constant.SignatoryStatusNotInvited,
		Email:  payload.Email,
	})
	if err != nil {
		return errors.New("failed to add signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	pbc.app.Logger.Infof("Paylod json : %s", string(data))
	var payload AnnotateSignatureUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureUpdate")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureUpdate}) {
		return errors.New("you do not have permission to update signature annotate")
	}

	err := pbc.app.Repository.SignatureAnnotate.Update(ctx, tx, map[string]any{
		"id":         payload.ID,
		"page":       uint(payload.Page),
		"x":          payload.X,
		"y":          payload.Y,
		"width":      payload.Width,
		"height":     payload.Height,
		"color":      payload.Color,
		"project_id": project.ID,
	})
	if err != nil {
		return errors.New("failed to update signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureRemove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureRemove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureRemove")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureRemove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureRemove}) {
		return errors.New("you do not have permission to remove signature annotate")
	}

	err := pbc.app.Repository.SignatureAnnotate.Delete(ctx, tx, payload.ID)
	if err != nil {
		return errors.New("failed to remove signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureInvite(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureInvite
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureInvite")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureInvite: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureInvite}) {
		return errors.New("you do not have permission to invite signature annotate")
	}

	err := pbc.app.Repository.SignatureAnnotate.InviteSignatory(ctx, tx, payload.ID)
	if err != nil {
		return errors.New("failed to invite signatory")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureApprove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureApprove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureApprove")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureApprove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureApprove}) {
		return errors.New("you do not have permission to approve signature")
	}

	// TODO: implement approve signature by accept file

	return nil
}

func (pbc ProjectBuilderController) handleSettingsUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	// print json of data
	pbc.app.Logger.Debugf("SettingsUpdate: %s", string(data))

	var payload SettingsUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for SettingsUpdate")
	}
	pbc.app.Logger.Debugf("SettingsUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.SettingsUpdate}) {
		return errors.New("you do not have permission to update settings")
	}

	log.Printf("QrCodeEnabled: %t", payload.QrCodeEnabled)

	err := pbc.app.Repository.Project.UpdateSetting(ctx, tx, project.ID, payload.QrCodeEnabled)
	if err != nil {
		return errors.New("failed to update project settings")
	}

	return nil
}

func (pbc ProjectBuilderController) handleTableUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload TableUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for TableUpdate")
	}

	file, err := ctx.FormFile("csvFile")
	if err != nil {
		pbc.app.Logger.Errorf("Failed to get csv file: %v", err)
		return errors.New("failed to get csv file")
	}
	if file == nil {
		return errors.New("csv file is required")
	}
	payload.CSVFile = file

	pbc.app.Logger.Debugf("TableUpdate: %+v", payload.CSVFile.Filename)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.TableUpdate}) {
		return errors.New("you do not have permission to update table")
	}

	f, err := payload.CSVFile.Open()
	if err != nil {
		return errors.New("failed to open csv file")
	}
	defer f.Close()

	// Create a temp file
	tmp, err := os.CreateTemp("", "autocert-*.csv")
	if err != nil {
		pbc.app.Logger.Errorf("Failed to create temp file: %v", err)
		return fmt.Errorf("failed to save table data")
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, f); err != nil {
		pbc.app.Logger.Errorf("Failed to copy file: %v", err)
		return fmt.Errorf("failed to save table data")
	}
	if err := tmp.Close(); err != nil {
		pbc.app.Logger.Errorf("Failed to close temp file: %v", err)
		return fmt.Errorf("failed to save table data")
	}

	_, err = autocert.ReadCSVFromFile(tmp.Name())
	if err != nil {
		return errors.New("invalid csv file")
	}

	info, err := pbc.uploadFileToS3ByPath(tmp.Name())
	if err != nil {
		pbc.app.Logger.Warnf("Failed to upload csv file: %v", err)
		return errors.New("failed to upload csv file")
	}

	err = pbc.app.Repository.Project.UpdateCSVFile(ctx, tx, *project, &model.File{
		FileName:       filepath.Base(tmp.Name()),
		UniqueFileName: info.Key,
		BucketName:     info.Bucket,
		Size:           info.Size,
	})
	if err != nil {
		pbc.app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{})

		pbc.app.Logger.Warnf("Failed to update project table: %v", err)
		return errors.New("failed to update project table")
	}

	return nil
}
