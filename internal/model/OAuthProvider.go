package model

type OAuthProvider struct {
	BaseModel
	ProviderType   string `gorm:"type:varchar(50);not null;" json:"providerType" form:"providerType" binding:"required"`
	ProviderUserId string `gorm:"unique;not null;type:text" json:"providerUserId" form:"providerUserId" binding:"required"`
	AccessToken    string `gorm:"type:text; default:null" json:"access_token" form:"access_token"`
	RefreshToken   string `gorm:"type:text; default:null" json:"refresh_token" form:"refresh_token"`
	UserID         string `gorm:"type:text;not null" json:"user_id" form:"user_id"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user" form:"user"`
}

func (op OAuthProvider) TableName() string {
	return "oauth_providers"
}
