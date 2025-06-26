package model

import "github.com/SeakMengs/AutoCert/pkg/autocert"

type Certificate struct {
	BaseModel
	Number            int                      `gorm:"type:int;not null" json:"number" form:"number"`
	CertificateFileId string                   `gorm:"type:text;not null" json:"certificateFileId" form:"certificateFileId" binding:"required"`
	ProjectID         string                   `gorm:"type:text;not null" json:"projectId" form:"projectId"`
	Type              autocert.CertificateType `gorm:"type:integer;default:0" json:"type" form:"type"`

	CertificateFile File    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"certificateFile,omitempty" form:"certificateFile"`
	Project         Project `gorm:"constraint:OnDelete:SET NULL" json:"project" form:"project"`
}

func (c Certificate) TableName() string {
	return "certificates"
}
