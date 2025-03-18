package model

type SignatureAnnotate struct {
	BaseAnnotateModel
	BaseModel

	SignatureData string `gorm:"type:text" json:"signature_data" form:"signature_data"`
	Email         string `gorm:"type:citext;not null" json:"email" form:"email" binding:"required"`
}

func (sa SignatureAnnotate) TableName() string {
	return "signature_annotates"
}
