package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/mailer"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/queue"
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
// TODO: check lock like frontend
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
	ID       string `json:"id" binding:"required" form:"id"`
	SendMail bool   `json:"sendMail" form:"sendMail"`
}

type AnnotateSignatureReject struct {
	ID     string `json:"id" binding:"required" form:"id"`
	Reason string `json:"reason" binding:"required" form:"reason"`
}

type AnnotateSignatureApprove struct {
	ID            string                `json:"id" binding:"required" form:"id"`
	SignatureFile *multipart.FileHeader `json:"signatureFile" binding:"required" form:"signatureFile"`
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

	eventsJSON := ctx.PostForm("events")
	if eventsJSON == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("events is required"), "events"), nil)
		return
	}

	var events []AutoCertChangeEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, ErrFailedToUpdateProjectBuilder, util.GenerateErrorMessages(errors.New("failed to parse events"), "events"), nil)
		return
	}

	var (
		addEvents         []AutoCertChangeEvent
		updateEvents      []AutoCertChangeEvent
		removeEvents      []AutoCertChangeEvent
		otherEvents       []AutoCertChangeEvent
		tableUpdateEvents []AutoCertChangeEvent
	)

	for _, event := range events {
		switch event.Type {
		case constant.TableUpdate:
			tableUpdateEvents = append(tableUpdateEvents, event)
		case constant.AnnotateColumnAdd, constant.AnnotateSignatureAdd:
			addEvents = append(addEvents, event)
		case constant.AnnotateColumnUpdate, constant.AnnotateSignatureUpdate, constant.SettingsUpdate, constant.AnnotateSignatureInvite, constant.AnnotateSignatureApprove, constant.AnnotateSignatureReject:
			updateEvents = append(updateEvents, event)
		case constant.AnnotateColumnRemove, constant.AnnotateSignatureRemove:
			removeEvents = append(removeEvents, event)
		default:
			otherEvents = append(otherEvents, event)
		}
	}

	// sort events by type such that the events are processed in the correct order
	// sort events by making table update last since it involve updating file which is not transactional
	events = slices.Clone(addEvents)
	events = append(events, updateEvents...)
	events = append(events, otherEvents...)
	events = append(events, removeEvents...)
	events = append(events, tableUpdateEvents...)

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

// TODO: add rerturn error key and defer function, onError of each event handler
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
		constant.AnnotateSignatureReject:  pbc.handleAnnotateSignatureReject,
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

	annot, err := pbc.app.Repository.SignatureAnnotate.GetById(ctx, tx, payload.ID, project.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("signature annotate not found")
		}
		return errors.New("failed to get signature annotate")
	}

	err = pbc.app.Repository.SignatureAnnotate.Update(ctx, tx, map[string]any{
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

	user, err := pbc.getAuthUser(ctx)
	if err != nil {
		return errors.New("failed to get authenticated user")
	}
	err = pbc.app.Repository.ProjectLog.Save(ctx, tx, &model.ProjectLog{
		Role:      user.Email,
		ProjectID: project.ID,
		Action:    "Requestor updated signature annotate",
		Description: fmt.Sprintf(
			"%s has updated the signature annotate. Signature id: %s. Details: status %s, page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			annot.Email, annot.ID, util.GetSignatureStatus(annot.Status), annot.Page, annot.X, annot.Y, annot.Width, annot.Height,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		pbc.app.Logger.Errorf("Failed to save project log: %v", err)
		return errors.New("failed to log project activity")
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

	annot, err := pbc.app.Repository.SignatureAnnotate.GetById(ctx, tx, payload.ID, project.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("signature annotate not found")
		}
		return errors.New("failed to get signature annotate")
	}

	err = pbc.app.Repository.SignatureAnnotate.Delete(ctx, tx, payload.ID)
	if err != nil {
		return errors.New("failed to remove signature annotate")
	}

	user, err := pbc.getAuthUser(ctx)
	if err != nil {
		return errors.New("failed to get authenticated user")
	}
	err = pbc.app.Repository.ProjectLog.Save(ctx, tx, &model.ProjectLog{
		Role:      user.Email,
		ProjectID: project.ID,
		Action:    "Requestor removed signature annotate",
		Description: fmt.Sprintf(
			"%s has removed the signature annotate. Signature id: %s. Details: status %s, page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			annot.Email, annot.ID, util.GetSignatureStatus(annot.Status), annot.Page, annot.X, annot.Y, annot.Width, annot.Height,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		pbc.app.Logger.Errorf("Failed to save project log: %v", err)
		return errors.New("failed to log project activity")
	}

	// TODO: remove signature file if exist

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

	annot, err := pbc.app.Repository.SignatureAnnotate.GetById(ctx, tx, payload.ID, project.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("signature annotate not found")
		}
		return errors.New("failed to get signature annotate")
	}

	if annot.Status != constant.SignatoryStatusNotInvited {
		return errors.New("signature annotate is not in the correct status to invite signatory")
	}

	err = pbc.app.Repository.SignatureAnnotate.InviteSignatory(ctx, tx, payload.ID)
	if err != nil {
		return errors.New("failed to invite signatory")
	}

	user, err := pbc.getAuthUser(ctx)
	if err != nil {
		return errors.New("failed to get authenticated user")
	}

	err = pbc.app.Repository.ProjectLog.Save(ctx, tx, &model.ProjectLog{
		Role:      user.Email,
		ProjectID: project.ID,
		Action:    "Requestor invited signatory",
		Description: fmt.Sprintf(
			"%s has been invited to sign the certificate. Signature id: %s. Details: page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			annot.Email, annot.ID, annot.Page, annot.X, annot.Y, annot.Width, annot.Height,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		pbc.app.Logger.Errorf("Failed to save project log: %v", err)
		return errors.New("failed to log project activity")
	}

	if payload.SendMail {
		recipientName := annot.Email
		if atIdx := strings.Index(recipientName, "@"); atIdx > 0 {
			recipientName = recipientName[:atIdx]
		}

		mailData, err := queue.NewSignatureRequestInvitationMailJob(annot.Email,
			mailer.SignatureRequestInvitationData{
				RecipientName:           recipientName,
				InviterName:             fmt.Sprintf("%s (%s)", user.LastName, user.Email),
				CertificateProjectTitle: project.Title,
				SigningURL:              fmt.Sprintf("%s/dashboard/projects/%s/builder", pbc.app.Config.FRONTEND_URL, project.ID),
				APP_NAME:                util.GetAppName(),
				APP_LOGO_URL:            util.GetAppLogoURL(pbc.app.Config.FRONTEND_URL),
				ProjectID:               project.ID,
				SignatureRequestID:      annot.ID,
			})
		if err != nil {
			pbc.app.Logger.Errorf("Failed to create signature request invitation mail job: %v", err)
			return errors.New("failed to create signature request invitation mail job")
		}

		payloadBytes, err := json.Marshal(mailData)
		if err != nil {
			pbc.app.Logger.Errorf("Failed to marshal signature request invitation mail job payload: %v", err)
			return errors.New("failed to create signature request invitation mail job")
		}

		if err := pbc.app.Queue.Publish(queue.QueueMail, payloadBytes); err != nil {
			pbc.app.Logger.Errorf("Failed to publish signature request invitation mail job: %v", err)
			return errors.New("failed to create signature request invitation mail job")
		}
	} else {
		pbc.app.Logger.Debugf("Skip sending mail for signature invite for signature id: %s and project id: %s because SendMail is false", annot.ID, project.ID)
	}

	return nil
}

func (pbc ProjectBuilderController) handleAnnotateSignatureReject(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureReject
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureReject")
	}
	pbc.app.Logger.Debugf("AnnotateSignatureReject: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureReject}) {
		return errors.New("you do not have permission to reject signature annotate")
	}

	annot, err := pbc.app.Repository.SignatureAnnotate.GetById(ctx, tx, payload.ID, project.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("signature annotate not found")
		}
		return errors.New("failed to get signature annotate")
	}

	err = pbc.app.Repository.SignatureAnnotate.RejectSignature(ctx, tx, payload.ID, payload.Reason)
	if err != nil {
		return errors.New("failed to reject signatory")
	}

	user, err := pbc.getAuthUser(ctx)
	if err != nil {
		return errors.New("failed to get authenticated user")
	}

	var description string
	if len(payload.Reason) > 0 {
		description = fmt.Sprintf(
			"%s has rejected the signature request with reason: %s. Signature id: %s. Details: page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			annot.Email, payload.Reason, annot.ID, annot.Page, annot.X, annot.Y, annot.Width, annot.Height,
		)
	} else {
		description = fmt.Sprintf(
			"%s has rejected the signature request. Signature id: %s. Details: page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			annot.Email, annot.ID, annot.Page, annot.X, annot.Y, annot.Width, annot.Height,
		)
	}

	err = pbc.app.Repository.ProjectLog.Save(ctx, tx, &model.ProjectLog{
		Role:        user.Email,
		ProjectID:   project.ID,
		Action:      "Signatory reject signature request",
		Description: description,
		Timestamp:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		pbc.app.Logger.Errorf("Failed to save project log: %v", err)
		return errors.New("failed to log project activity")
	}

	return nil
}

func (pbc ProjectBuilderController) handleSettingsUpdate(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload SettingsUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for SettingsUpdate")
	}
	pbc.app.Logger.Debugf("SettingsUpdate: %+v", payload)

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.SettingsUpdate}) {
		return errors.New("you do not have permission to update settings")
	}

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
	tmp, err := util.CreateTemp("autocert-*.csv")
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

	records, err := autocert.ReadCSVFromFile(tmp.Name())
	if err != nil {
		return errors.New("invalid csv file")
	}
	csvData, err := autocert.ParseCSVToMap(records)
	if err != nil {
		pbc.app.Logger.Errorf("Failed to parse csv file: %v", err)
		return errors.New("invalid csv file")
	}

	if len(csvData) > pbc.app.Config.APP.MAX_CERTIFICATES_PER_PROJECT {
		pbc.app.Logger.Warnf("CSV file exceeds maximum number of certificates: %d", len(csvData))
		return fmt.Errorf("csv file exceeds maximum number of certificates: %d", pbc.app.Config.APP.MAX_CERTIFICATES_PER_PROJECT)
	}

	info, err := util.UploadFileToS3ByPath(tmp.Name(), &util.FileUploadOptions{
		DirectoryPath: util.GetProjectDirectoryPath(project.ID),
		UniquePrefix:  true,
		Bucket:        pbc.app.Config.Minio.BUCKET,
		S3:            pbc.app.S3,
	})
	if err != nil {
		pbc.app.Logger.Warnf("Failed to upload csv file: %v", err)
		return errors.New("failed to upload csv file")
	}

	err = pbc.app.Repository.Project.UpdateCSVFile(ctx, tx, *project, &model.File{
		FileName:       util.ToProjectDirectoryPath(project.ID, tmp.Name()),
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

func (pbc ProjectBuilderController) handleAnnotateSignatureApprove(ctx *gin.Context, tx *gorm.DB, roles []constant.ProjectRole, project *model.Project, data json.RawMessage) error {
	var payload AnnotateSignatureApprove
	if err := json.Unmarshal(data, &payload); err != nil {
		return errors.New("invalid payload for AnnotateSignatureApprove")
	}

	sigFile, err := ctx.FormFile(fmt.Sprintf("signature_approve_file_%s", payload.ID))
	if err != nil {
		pbc.app.Logger.Errorf("Failed to approve signature: cannot get signature file for annotate id %s: %v", payload.ID, err)
		return fmt.Errorf("failed to get signature file for annotate id %s", payload.ID)
	}

	if sigFile == nil {
		return errors.New("signature file is required")
	}

	ext := filepath.Ext(sigFile.Filename)
	if !slices.Contains(ALLOWED_SIGNATURE_FILE_TYPE, ext) {
		pbc.app.Logger.Errorf("Failed to approve signature: invalid file type %s", ext)
		return errors.New("invalid file type")
	}

	if !util.HasPermission(roles, []constant.ProjectPermission{constant.AnnotateSignatureApprove}) {
		return errors.New("you do not have permission to approve signature")
	}

	user, err := pbc.getAuthUser(ctx)
	if err != nil {
		pbc.app.Logger.Errorf("Failed to get auth user: %v", err)
		return errors.New("failed to get authenticated user")
	}

	sa, err := pbc.app.Repository.SignatureAnnotate.GetById(ctx, nil, payload.ID, project.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("annotate signature not found")
		}
		return errors.New("failed to get signature annotate")
	}

	if !strings.EqualFold(sa.Email, user.Email) {
		return errors.New("the signature cannot be approved because it is not assigned to you")
	}

	if sa.Status != constant.SignatoryStatusInvited {
		return errors.New("the signature cannot be approved because it is not in the invited status")
	}

	info, err := util.UploadFileToS3ByFileHeader(sigFile, &util.FileUploadOptions{
		DirectoryPath: util.GetProjectDirectoryPath(project.ID),
		UniquePrefix:  true,
		Bucket:        pbc.app.Config.Minio.BUCKET,
		S3:            pbc.app.S3,
	})
	if err != nil {
		return errors.New("failed to upload signature file")
	}

	err = pbc.app.Repository.SignatureAnnotate.ApproveSignature(ctx, tx, payload.ID, &model.File{
		FileName:       util.ToProjectDirectoryPath(project.ID, sigFile.Filename),
		UniqueFileName: info.Key,
		BucketName:     info.Bucket,
		Size:           info.Size,
	})
	if err != nil {
		if deleteErr := pbc.app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{}); deleteErr != nil {
			pbc.app.Logger.Errorf("Failed to delete file after approval failure: %v", deleteErr)
		}
		return errors.New("failed to approve signature")
	}

	err = pbc.app.Repository.ProjectLog.Save(ctx, tx, &model.ProjectLog{
		Role:      user.Email,
		ProjectID: project.ID,
		Action:    "Signatory approved signature",
		Description: fmt.Sprintf(
			"%s has approved the signature. Signature id: %s. Details: page %d, position (%.2f, %.2f), size (%.2f, %.2f).",
			sa.Email, sa.ID, sa.Page, sa.X, sa.Y, sa.Width, sa.Height,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		pbc.app.Logger.Errorf("Failed to save project log: %v", err)
		return errors.New("failed to log project activity")
	}

	return nil
}
