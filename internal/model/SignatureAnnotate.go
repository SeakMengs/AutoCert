package model

import (
	"context"
	"os"
	"path/filepath"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/minio/minio-go/v7"
)

type SignatureAnnotate struct {
	BaseAnnotateModel
	BaseModel

	Status          constant.SignatoryStatus `gorm:"type:integer;default:0" json:"status" form:"status"`
	SignatureFileID string                   `gorm:"type:text;default:null" json:"-" form:"signatureFileId" binding:"required"`
	Email           string                   `gorm:"type:citext;not null" json:"email" form:"email" binding:"required"`

	SignatureFile File `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-" form:"-"`
}

func (sa SignatureAnnotate) TableName() string {
	return "signature_annotates"
}

// Don't forget to defer remove the file after using the temp file
func (sa SignatureAnnotate) ToAutoCertSignatureAnnotate(ctx context.Context, s3 *minio.Client) (*autocert.SignatureAnnotate, error) {
	ext := filepath.Ext(sa.SignatureFile.FileName)
	tmp, err := os.CreateTemp("", "signature_file_*"+ext)
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
