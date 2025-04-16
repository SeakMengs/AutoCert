package model

type BaseAnnotateModel struct {
	Page   uint    `gorm:"type:integer;not null" json:"page" form:"page" binding:"required"`
	X      float64 `gorm:"type:double precision;not null" json:"x" form:"x" binding:"required"`
	Y      float64 `gorm:"type:double precision;not null" json:"y" form:"y" binding:"required"`
	Width  float64 `gorm:"type:double precision;not null" json:"width" form:"width" binding:"required"`
	Height float64 `gorm:"type:double precision;not null" json:"height" form:"height" binding:"required"`
	Color  string  `gorm:"type:varchar(20)" json:"color" form:"color" binding:"required"`

	ProjectID string  `gorm:"type:text;not null" json:"projectId" form:"projectId"`
	Project   Project `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"project" form:"project"`
}
