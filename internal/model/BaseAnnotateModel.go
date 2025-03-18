package model

type BaseAnnotateModel struct {
	Page   uint    `gorm:"type:integer;not null" json:"page" form:"page"`
	X      float64 `gorm:"type:double precision;not null" json:"x" form:"x"`
	Y      float64 `gorm:"type:double precision;not null" json:"y" form:"y"`
	Width  float64 `gorm:"type:double precision;not null" json:"width" form:"width"`
	Height float64 `gorm:"type:double precision;not null" json:"height" form:"height"`
	Color  string  `gorm:"type:varchar(20)" json:"color" form:"color"` // Hex color code

	ProjectID string  `gorm:"type:text;not null" json:"project_id" form:"project_id"`
	Project   Project `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"project" form:"project"`
}
