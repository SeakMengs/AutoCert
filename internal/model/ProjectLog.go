package model

type ProjectLogs struct {
	BaseModel

	Role        string `gorm:"type:text;not null;" json:"role" form:"role" binding:"required"`
	Action      string `gorm:"type:text;not null;" json:"action" form:"action" binding:"required"`
	Description string `gorm:"type:text;not null;" json:"description" form:"description" binding:"required"`
	Timestamp   string `gorm:"type:timestamp;not null;" json:"timestamp" form:"timestamp" binding:"required"`

	ProjectID string  `gorm:"type:text;not null" json:"projectId" form:"projectId"`
	Project   Project `gorm:"constraint:OnDelete:SET NULL" json:"project" form:"project"`
}

func (pl ProjectLogs) TableName() string {
	return "project_logs"
}
