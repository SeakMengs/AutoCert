package model

type Project struct {
	BaseModel
	Title        string `gorm:"type:varchar(30);not null;" json:"title" form:"title" binding:"required"`
	TemplateFile string `gorm:"type:text;not null;" json:"templateFile" form:"templateFile" binding:"required"`
	IsPublic     bool   `gorm:"type:boolean;default:false" json:"isPublic" form:"isPublic"`
	EmbedQr      bool   `gorm:"type:boolean;default:false" json:"embedQr" form:"embedQr"`

	UserID string `gorm:"type:text;not null" json:"user_id" form:"user_id"`
	User   User   `json:"user" form:"user"`
}

func (p Project) TableName() string {
	return "projects"
}
