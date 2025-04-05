package model

type OAuthProvider struct {
	BaseModel
	ProviderType   string `gorm:"type:varchar(50);not null;" json:"providerType" form:"providerType" binding:"required"`
	ProviderUserId string `gorm:"unique;not null;type:text" json:"providerUserId" form:"providerUserId" binding:"required"`
	AccessToken    string `gorm:"type:text; default:null" json:"accessToken" form:"accessToken"`
	RefreshToken   string `gorm:"type:text; default:null" json:"refreshToken" form:"refreshToken"`
	UserID         string `gorm:"type:text;not null" json:"userId" form:"userId"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user" form:"user"`
}

func (op OAuthProvider) TableName() string {
	return "oauth_providers"
}
