package model

type ColumnAnnotate struct {
	BaseAnnotateModel
	BaseModel

	Value               string  `gorm:"type:varchar(200)" json:"value" form:"value" binding:"required"`
	FontName            string  `gorm:"type:varchar(200)" json:"fontName" form:"fontName"`
	FontSize            float64 `gorm:"type:double precision;not null" json:"fontSize" form:"fontSize"`
	FontWeight          string  `gorm:"type:varchar(50)" json:"fontWeight" form:"fontWeight"`
	FontColor           string  `gorm:"type:varchar(20)" json:"fontColor" form:"fontColor"`
	TextFitRectangleBox bool    `gorm:"type:boolean;default:true" json:"textFitRectangleBox" form:"textFitRectangleBox"`
}

func (ca ColumnAnnotate) TableName() string {
	return "column_annotates"
}
