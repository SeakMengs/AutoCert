package model

type User struct {
	BaseModel
	Email      string `gorm:"unique;not null;type:citext" json:"email" form:"email" binding:"required"`
	FirstName  string `gorm:"type:varchar(30);not null;" json:"firstName" form:"firstName" binding:"required"`
	LastName   string `gorm:"type:varchar(30);not null;" json:"lastName" form:"lastName" binding:"required"`
	ProfileURL string `gorm:"type:text;default:null" json:"profileURL" form:"profileURL"`
}

func (u User) TableName() string {
	return "users"
}
