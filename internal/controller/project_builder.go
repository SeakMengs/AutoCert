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
)

type ProjectBuilderController struct {
	*baseController
}

type EventType string

const (
	EventAnnotateColumnAdd        EventType = "annotate:column:add"
	EventAnnotateColumnUpdate     EventType = "annotate:column:update"
	EventAnnotateColumnRemove     EventType = "annotate:column:remove"
	EventAnnotateSignatureAdd     EventType = "annotate:signature:add"
	EventAnnotateSignatureUpdate  EventType = "annotate:signature:update"
	EventAnnotateSignatureRemove  EventType = "annotate:signature:remove"
	EventAnnotateSignatureInvite  EventType = "annotate:signature:invite"
	EventAnnotateSignatureApprove EventType = "annotate:signature:approve"
	EventSettingsUpdate           EventType = "settings:update"
	EventTableUpdate              EventType = "table:update"
)

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
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ColumnAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateColumnUpdate struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ColumnAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateColumnRemove struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureAdd struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		SignatureAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureUpdate struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		SignatureAnnotateState
		Page int `json:"page" binding:"required" form:"page"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureRemove struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureInvite struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ID    string `json:"id" binding:"required" form:"id"`
		Email string `json:"email" binding:"required,email" form:"email"`
	} `json:"data" binding:"required" form:"data"`
}

type AnnotateSignatureApprove struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		ID string `json:"id" binding:"required" form:"id"`
	} `json:"data" binding:"required" form:"data"`
}

type SettingsUpdate struct {
	Type EventType        `json:"type" binding:"required" form:"type"`
	Data AutoCertSettings `json:"data" binding:"required" form:"data"`
}

type TableUpdate struct {
	Type EventType `json:"type" binding:"required" form:"type"`
	Data struct {
		CSVFile *multipart.FileHeader `form:"csvFile" binding:"required"`
	} `json:"data" binding:"required" form:"data"`
}

// AutoCertChangeEvent is a generic wrapper that holds the event type and raw payload.
type AutoCertChangeEvent struct {
	Type EventType       `json:"type" binding:"required" form:"type"`
	Data json.RawMessage `json:"data" binding:"required" form:"data"`
}

func (pbc ProjectBuilderController) ProjectBuilder(ctx *gin.Context) {
	projectId := ctx.Param("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("projectId is required"), "projectId"), nil)
		return
	}

	roles, project, err := pbc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err), nil)
		return
	}

	if project != nil {
		if project.Status != constant.ProjectStatusPreparing {
			util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("project is not in preparing status"), "project"), nil)
			return
		}
	}

	var hasPermission bool

	for _, role := range roles {
		if role == constant.ProjectRoleOwner {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		util.ResponseFailed(ctx, http.StatusForbidden, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("user does not have permission"), "permission"), nil)
		return
	}

	// Get the events JSON from the form.
	eventsJSON := ctx.PostForm("events")
	if eventsJSON == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("events is required"), "events"), nil)
		return
	}

	// Unmarshal the events JSON into an array.
	var events []AutoCertChangeEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("failed to parse events"), "events"), nil)
		return
	}

	tx := pbc.app.Repository.DB.Begin()
	defer tx.Commit()

	// Process each event.
	for idx, event := range events {
		pbc.app.Logger.Debugf("Processing event #%d, type %s", idx, event.Type)
		switch event.Type {
		case EventAnnotateColumnAdd:
			var payload AnnotateColumnAdd
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateColumnAdd"), "events"), nil)
				return
			}
			pbc.app.Logger.Debugf("AnnotateColumnAdd: %+v", payload)

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
				tx.Rollback()
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to patch project builder", util.GenerateErrorMessages(err), nil)
				return
			}
		case EventAnnotateColumnUpdate:
			var payload AnnotateColumnUpdate
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateColumnUpdate"), "events"), nil)
				return
			}
			pbc.app.Logger.Debugf("AnnotateColumnUpdate: %+v", payload)

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
				tx.Rollback()
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to patch project builder", util.GenerateErrorMessages(err), nil)
				return
			}
		case EventAnnotateColumnRemove:
			var payload AnnotateColumnRemove
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateColumnRemove"), "events"), nil)
				return
			}
			pbc.app.Logger.Debugf("AnnotateColumnRemove: %+v", payload)

			err := pbc.app.Repository.ColumnAnnotate.Delete(ctx, tx, payload.Data.ID)
			if err != nil {
				tx.Rollback()
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to patch project builder", util.GenerateErrorMessages(err), nil)
				return
			}
		case EventAnnotateSignatureAdd:
			var payload AnnotateSignatureAdd
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateSignatureAdd"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("AnnotateSignatureAdd: %+v", payload)
		case EventAnnotateSignatureUpdate:
			var payload AnnotateSignatureUpdate
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateSignatureUpdate"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("AnnotateSignatureUpdate: %+v", payload)
		case EventAnnotateSignatureRemove:
			var payload AnnotateSignatureRemove
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateSignatureRemove"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("AnnotateSignatureRemove: %+v", payload)
		case EventAnnotateSignatureInvite:
			var payload AnnotateSignatureInvite
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateSignatureInvite"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("AnnotateSignatureInvite: %+v", payload)
		case EventAnnotateSignatureApprove:
			var payload AnnotateSignatureApprove
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for AnnotateSignatureApprove"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("AnnotateSignatureApprove: %+v", payload)
		case EventSettingsUpdate:
			var payload SettingsUpdate
			if err := json.Unmarshal(event.Data, &payload.Data); err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("invalid payload for SettingsUpdate"), "events"), nil)
				return
			}

			pbc.app.Logger.Debugf("SettingsUpdate: %+v", payload)
		case EventTableUpdate:
			file, err := ctx.FormFile("csvFile")
			if err != nil {
				util.ResponseFailed(ctx, http.StatusBadRequest, "Failed to patch project builder", util.GenerateErrorMessages(errors.New("failed to get csvFile"), "csvFile"), nil)
				return
			}

			pbc.app.Logger.Debugf("TableUpdate event: Received file %s", file.Filename)
		default:
			pbc.app.Logger.Errorf("Unknown event type: %s", event.Type)
		}
	}

	util.ResponseSuccess(ctx, nil)
}
