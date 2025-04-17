package model

import "github.com/SeakMengs/AutoCert/internal/constant"

type Project struct {
	BaseModel
	Title          string                 `gorm:"type:varchar(30);not null;" json:"title" form:"title" binding:"required"`
	IsPublic       bool                   `gorm:"type:boolean;default:false" json:"isPublic" form:"isPublic"`
	EmbedQr        bool                   `gorm:"type:boolean;default:false" json:"embedQr" form:"embedQr"`
	Status         constant.ProjectStatus `gorm:"type:integer;default:0" json:"status" form:"status"`
	TemplateFileID string                 `gorm:"type:text;not null" json:"templateFileId" form:"templateFileId" binding:"required"`
	CSVFileID      string                 `gorm:"type:text;default:null" json:"csvFileId" form:"csvFileId"`
	UserID         string                 `gorm:"type:text;not null" json:"userId" form:"userId"`

	TemplateFile       File                `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"templateFile,omitempty" form:"templateFile"`
	CSVFile            File                `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"csvFile,omitempty" form:"csvFile"`
	User               User                `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty" form:"user"`
	SignatureAnnotates []SignatureAnnotate `json:"signatureAnnotates,omitempty" form:"signatureAnnotates"`
	ColumnAnnotates    []ColumnAnnotate    `json:"columnAnnotates,omitempty" form:"columnAnnotates"`
}

func (p Project) TableName() string {
	return "projects"
}
