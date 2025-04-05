package model

type Project struct {
	BaseModel
	Title          string `gorm:"type:varchar(30);not null;" json:"title" form:"title" binding:"required"`
	IsPublic       bool   `gorm:"type:boolean;default:false" json:"isPublic" form:"isPublic"`
	EmbedQr        bool   `gorm:"type:boolean;default:false" json:"embedQr" form:"embedQr"`
	TemplateFileID string `gorm:"type:text;not null" json:"templateFileId" form:"templateFileId" binding:"required"`

	TemplateFile File   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"templateFile" form:"templateFile"`
	UserID       string `gorm:"type:text;not null" json:"userId" form:"userId"`
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user" form:"user"`
}

func (p Project) TableName() string {
	return "projects"
}
