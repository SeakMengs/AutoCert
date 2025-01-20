package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID         string `gorm:"type:text;primaryKey" json:"id"`
	Email      string `gorm:"unique;not null;type:citext" json:"email" form:"email" binding:"required"`
	FirstName  string `gorm:"type:varchar(30);not null;" json:"firstName" form:"firstName" binding:"required"`
	LastName   string `gorm:"type:varchar(30);not null;" json:"lastName" form:"lastName" binding:"required"`
	ProfileURL string `gorm:"type:text;default:null" json:"profileURL" form:"profileURL"`

	BaseModel
}

func (u User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// UUID version 4
	u.ID = uuid.NewString()
	return
}
