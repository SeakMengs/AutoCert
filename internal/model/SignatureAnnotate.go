package model

import "github.com/SeakMengs/AutoCert/internal/constant"

type SignatureAnnotate struct {
	BaseAnnotateModel
	BaseModel

	Status          constant.SignatoryStatus `gorm:"type:integer;default:0" json:"status" form:"status"`
	SignatureFileID string                   `gorm:"type:text" json:"signatureFileId" form:"signatureFileId" binding:"required"`
	SignatureFile   File                     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"signatureFile" form:"signatureFile"`
	Email           string                   `gorm:"type:citext;not null" json:"email" form:"email" binding:"required"`
}

func (sa SignatureAnnotate) TableName() string {
	return "signature_annotates"
}
