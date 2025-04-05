package model

type Token struct {
	BaseModel
	AccessToken  string `gorm:"type:text;default:null" json:"accessToken" form:"accessToken"`
	RefreshToken string `gorm:"type:text;default:null" json:"refreshToken" form:"refreshToken"`
	CanAccess    bool   `gorm:"not null;default:true" json:"canAccess" form:"canAccess"`
	CanRefresh   bool   `gorm:"not null;default:true" json:"canRefresh" form:"canRefresh"`

	UserID string `gorm:"type:text;not null" json:"userId" form:"userId"`
	User   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user" form:"user"`
}

func (t Token) TableName() string {
	return "tokens"
}
