package model

import (
	"context"
	"path/filepath"

	"github.com/SeakMengs/AutoCert/internal/constant"
	filestorage "github.com/SeakMengs/AutoCert/internal/file_storage"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

type SignatureAnnotate struct {
	BaseAnnotateModel
	BaseModel

	Status          constant.SignatoryStatus `gorm:"type:integer;default:0" json:"status" form:"status"`
	SignatureFileID string                   `gorm:"type:text;default:null" json:"-" form:"signatureFileId" binding:"required"`
	Email           string                   `gorm:"type:citext;not null" json:"email" form:"email" binding:"required"`
	Reason          string                   `gorm:"type:text;default:null" json:"reason" form:"reason"`

	SignatureFile File `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-" form:"-"`
}

func (sa SignatureAnnotate) TableName() string {
	return "signature_annotates"
}

// Don't forget to defer remove the file after using the temp file
func (sa SignatureAnnotate) ToAutoCertSignatureAnnotate(ctx context.Context, s3 *filestorage.MinioClient) (*autocert.SignatureAnnotate, error) {
	ext := filepath.Ext(sa.SignatureFile.FileName)
	tmp, err := util.CreateTemp("autocert_signature_file_*" + ext)
	if err != nil {
		return nil, err
	}

	err = sa.SignatureFile.DownloadToLocal(ctx, s3, tmp.Name())
	if err != nil {
		return nil, err
	}

	return &autocert.SignatureAnnotate{
		BaseAnnotate: autocert.BaseAnnotate{
			ID:       sa.ID,
			Type:     autocert.AnnotateTypeSignature,
			Position: autocert.Position{X: sa.X, Y: sa.Y},
			Size:     autocert.Size{Width: sa.Width, Height: sa.Height},
		},
		SignatureFilePath: tmp.Name(),
		Email:             sa.Email,
	}, nil
}
