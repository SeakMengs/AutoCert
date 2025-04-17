package controller

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectBuilderController struct {
	*baseController
}

type AnnotateType string

const (
	AnnotateTypeColumn    AnnotateType = "column"
	AnnotateTypeSignature AnnotateType = "signature"
)

type ColumnAnnotateState struct {
	model.ColumnAnnotate
	Type AnnotateType `json:"type" binding:"required"`
}

type SignatureAnnotateState struct {
	model.SignatureAnnotate
	Type AnnotateType `json:"type" binding:"required"`
}

type AutoCertSettings struct {
	QrCodeEnabled bool `json:"qrCodeEnabled" binding:"required"`
}

type AnnotateColumnAdd struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ColumnAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateColumnUpdate struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ColumnAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateColumnRemove struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureAdd struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		SignatureAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureUpdate struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		SignatureAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureRemove struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureInvite struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ID    string `json:"id" binding:"required" form:"id"`
		Email string `json:"email" binding:"required,email" form:"email"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureApprove struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type SettingsUpdate struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data AutoCertSettings           `json:"data" binding:"required" form:"data"`
}

type TableUpdate struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data struct {
		CSVFile *multipart.FileHeader `form:"csvFile" binding:"required"`
	} `json:"data" binding:"required" form:"data"`
}

// AutoCertChangeEvent is a generic wrapper that holds the event type and raw payload.
type AutoCertChangeEvent struct {
	Type constant.ProjectPermission `json:"type" binding:"required" form:"type"`
	Data json.RawMessage            `json:"data" binding:"required" form:"data"`
}

const (
	ErrFailedToPatchProjectBuilder = "Failed to patch project builder"
)

// Take list of event types and their corresponding payloads, if at least one of the patch fail, will revert all changes and respond with error
func (pbc ProjectBuilderController) ProjectBuilder(ctx *gin.Context) {
	projectId := ctx.Param("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("projectId is required"), "projectId"), nil)
		return
	}

	roles, project, err := pbc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil {
		util.ResponseFailed(ctx, http.StatusNotFound, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("project not found"), "project"), nil)
		return
	}

	if project.Status != constant.ProjectStatusPreparing {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("project is not in preparing status"), "project"), nil)
		return
	}

	// Get the events JSON from the form.
	eventsJSON := ctx.PostForm("events")
	if eventsJSON == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("events is required"), "events"), nil)
		return
	}

	// Unmarshal the events JSON into an array.
	var events []AutoCertChangeEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("failed to parse events"), "events"), nil)
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
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(errors.New("panic occurred"), "events"), nil)
		}
	}()

	handlers := pbc.getEventHandlers()

	for idx, event := range events {
		pbc.app.Logger.Debugf("Processing event #%d, type %s", idx, event.Type)
		handler, ok := handlers[event.Type]

		if !ok {
			pbc.app.Logger.Errorf("Unknown event type: %s", event.Type)
			continue
		}

		if err := handler(ctx, tx, roles, projectId, event.Data); err != nil {
			tx.Rollback()
			pbc.app.Logger.Errorf("Failed to handle event %s: %v", event.Type, err)
			util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(err, nil, "events"), nil)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, ErrFailedToPatchProjectBuilder, util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, nil)
}

type EventHandlerType func(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error

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

func (pbc ProjectBuilderController) handleAnnotateColumnAdd(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateColumnAdd
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnAdd")
	}
	pbc.app.Logger.Debugf("AnnotateColumnAdd: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnAdd}) {
		return errors.New("you do not have permission to add column annotate")
	}

	err := pbc.app.Repository.ColumnAnnotate.Create(ctx, tx, model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: payload.Data.ID,
		},
		BaseAnnotateModel: model.BaseAnnotateModel{
			Page:      uint(payload.Data.Page),
			X:         payload.Data.X,
			Y:         payload.Data.Y,
			Width:     payload.Data.Width,
			Height:    payload.Data.Height,
			Color:     payload.Data.Color,
			ProjectID: projectId,
		},
		Value:          payload.Data.Value,
		FontName:       payload.Data.FontName,
		FontSize:       payload.Data.FontSize,
		FontColor:      payload.Data.FontColor,
		FontWeight:     payload.Data.FontWeight,
		TextFitRectBox: payload.Data.TextFitRectBox,
	})
	if err != nil {
		return errors.New("failed to add column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateColumnUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateColumnUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnUpdate")
	}
	pbc.app.Logger.Debugf("AnnotateColumnUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnUpdate}) {
		return errors.New("you do not have permission to update column annotate")
	}

	err := pbc.app.Repository.ColumnAnnotate.Update(ctx, tx, model.ColumnAnnotate{
		BaseModel: model.BaseModel{
			ID: payload.Data.ID,
		},
		BaseAnnotateModel: model.BaseAnnotateModel{
			Page:      uint(payload.Data.Page),
			X:         payload.Data.X,
			Y:         payload.Data.Y,
			Width:     payload.Data.Width,
			Height:    payload.Data.Height,
			Color:     payload.Data.Color,
			ProjectID: projectId,
		},
		Value:          payload.Data.Value,
		FontName:       payload.Data.FontName,
		FontSize:       payload.Data.FontSize,
		FontColor:      payload.Data.FontColor,
		FontWeight:     payload.Data.FontWeight,
		TextFitRectBox: payload.Data.TextFitRectBox,
	})
	if err != nil {
		return errors.New("failed to update column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateColumnRemove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateColumnRemove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateColumnRemove")
	}
	pbc.app.Logger.Debugf("AnnotateColumnRemove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateColumnRemove}) {
		return errors.New("you do not have permission to remove column annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureAdd(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateSignatureAdd
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureAdd")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureAdd: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureAdd}) {
		return errors.New("you do not have permission to add signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateSignatureUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureUpdate")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureUpdate}) {
		return errors.New("you do not have permission to update signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureRemove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateSignatureRemove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureRemove")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureRemove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureRemove}) {
		return errors.New("you do not have permission to remove signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureInvite(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateSignatureInvite
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureInvite")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureInvite: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureInvite}) {
		return errors.New("you do not have permission to invite signature annotate")
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureApprove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload AnnotateSignatureApprove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureApprove")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureApprove: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureApprove}) {
		return errors.New("you do not have permission to approve signature")
	}

	return nil
}

func (pbc ProjectBuilderController) handleSettingsUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload SettingsUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for SettingsUpdate")
	}
	pbc.app.Logger.Debugf("SettingsUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.SettingsUpdate}) {
		return errors.New("you do not have permission to update settings")
	}

	return nil
}

func (pbc ProjectBuilderController) handleTableUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, projectId string, data json.RawMessage) error {
	var payload TableUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for TableUpdate")
	}
	pbc.app.Logger.Debugf("TableUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.TableUpdate}) {
		return errors.New("you do not have permission to update table")
	}

	return nil
}
