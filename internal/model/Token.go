package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Token struct {
	ID           string `gorm:"type:text;primaryKey" json:"id"`
	UserID       string `gorm:"type:text;not null" json:"user_id"`
	AccessToken  string `gorm:"type:text;default:null" json:"access_token"`
	RefreshToken string `gorm:"type:text;default:null" json:"refresh_token"`
	CanAccess    bool   `gorm:"not null;default:true" json:"can_access"`
	CanRefresh   bool   `gorm:"not null;default:true" json:"can_refresh"`
	BaseModel

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user"`
}

func (t *Token) BeforeCreate(tx *gorm.DB) (err error) {
	// UUID version 4
	t.ID = uuid.NewString()
	return
}

func (t Token) TableName() string {
	return "tokens"
}
