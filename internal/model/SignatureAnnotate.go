package model

type SignatureAnnotate struct {
	BaseAnnotateModel
	BaseModel

	SignatureFileID string `gorm:"type:text;not null" json:"signatureFileId" form:"signatureFileId" binding:"required"`
	SignatureFile   File   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"signatureFile" form:"signatureFile"`
	Email           string `gorm:"type:citext;not null" json:"email" form:"email" binding:"required"`
}

func (sa SignatureAnnotate) TableName() string {
	return "signature_annotates"
}
