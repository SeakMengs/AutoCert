package model

import "github.com/SeakMengs/AutoCert/pkg/autocert"

type ColumnAnnotate struct {
	BaseAnnotateModel
	BaseModel

	Value          string  `gorm:"type:varchar(200)" json:"value" form:"value" binding:"required"`
	FontName       string  `gorm:"type:varchar(200)" json:"fontName" form:"fontName"`
	FontSize       float64 `gorm:"type:double precision;not null" json:"fontSize" form:"fontSize"`
	FontWeight     string  `gorm:"type:varchar(50)" json:"fontWeight" form:"fontWeight"`
	FontColor      string  `gorm:"type:varchar(20)" json:"fontColor" form:"fontColor"`
	TextFitRectBox bool    `gorm:"type:boolean;default:true" json:"textFitRectBox" form:"textFitRectBox"`
}

func (ca ColumnAnnotate) TableName() string {
	return "column_annotates"
}

func (ca ColumnAnnotate) ToAutoCertColumnAnnotate() *autocert.ColumnAnnotate {
	return &autocert.ColumnAnnotate{
		BaseAnnotate: autocert.BaseAnnotate{
			ID:       ca.ID,
			Type:     autocert.AnnotateTypeColumn,
			Position: autocert.Position{X: ca.X, Y: ca.Y},
			Size:     autocert.Size{Width: ca.Width, Height: ca.Height},
		},
		Value:          ca.Value,
		FontName:       ca.FontName,
		FontColor:      ca.FontColor,
		FontSize:       ca.FontSize,
		FontWeight:     autocert.FontWeight(ca.FontWeight),
		TextFitRectBox: ca.TextFitRectBox,
		// TODO: add text align to model
		TextAlign: autocert.TextAlignCenter,
	}
}
