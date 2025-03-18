package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string     `gorm:"type:text;primaryKey" json:"id"`
	CreatedAt *time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP;not null" json:"-"`
	UpdatedAt *time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP;onUpdate:CURRENT_TIMESTAMP;not null" json:"-"`
}

func (bm *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	// UUID version 4
	bm.ID = uuid.NewString()
	return
}
