package model

type Token struct {
	BaseModel
	AccessToken  string `gorm:"type:text;default:null" json:"access_token" form:"access_token"`
	RefreshToken string `gorm:"type:text;default:null" json:"refresh_token" form:"refresh_token"`
	CanAccess    bool   `gorm:"not null;default:true" json:"can_access" form:"can_access"`
	CanRefresh   bool   `gorm:"not null;default:true" json:"can_refresh" form:"can_refresh"`

	UserID string `gorm:"type:text;not null" json:"user_id" form:"user_id"`
	User   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user" form:"user"`
}

func (t Token) TableName() string {
	return "tokens"
}
