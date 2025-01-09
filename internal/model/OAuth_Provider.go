package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OAuthProvider struct {
	ID             string `gorm:"type:text;primaryKey" json:"id"`
	ProviderType   string `gorm:"type:varchar(50);not null;" json:"providerType"`
	ProviderUserId string `gorm:"unique;not null;type:text" json:"providerUserId"`
	AccessToken    string `gorm:"type:text; default:null" json:"access_token"`
	RefreshToken   string `gorm:"type:text; default:null" json:"refresh_token"`
	UserID         string `gorm:"type:text;not null" json:"user_id"`
	BaseModel

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user"`
}

func (op *OAuthProvider) BeforeCreate(tx *gorm.DB) (err error) {
	// UUID version 4
	op.ID = uuid.NewString()
	return
}

func (op OAuthProvider) TableName() string {
	return "oauth_providers"
}
