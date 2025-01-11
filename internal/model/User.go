package model

import (
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string `gorm:"type:text;primaryKey" json:"id"`
	Email     string `gorm:"unique;not null;type:citext" json:"email" form:"email" binding:"required"`
	Password  string `gorm:"type:text;not null;" json:"-" form:"password" binding:"required"`
	FirstName string `gorm:"type:varchar(30);not null;" json:"firstName" form:"firstName" binding:"required"`
	LastName  string `gorm:"type:varchar(30);not null;" json:"lastName" form:"lastName" binding:"required"`

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

func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	if u.Password != "" {
		hashedPassword, err := util.HashPassword(u.Password)
		if err != nil {
			return err
		}

		u.Password = hashedPassword
	}

	return nil
}
