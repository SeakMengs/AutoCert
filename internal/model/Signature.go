package model

type Signature struct {
	BaseModel
	UserID          string `gorm:"type:text;not null" json:"userId" form:"userId"`
	SignatureFileID string `gorm:"type:text;not null" json:"-" form:"signatureFileId" binding:"required"`

	User          User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-" form:"-"`
	SignatureFile File `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-" form:"-"`
}

func (s Signature) TableName() string {
	return "signatures"
}
