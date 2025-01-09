package model

import "time"

type BaseModel struct {
	CreatedAt *time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP;not null" json:"-"`
	UpdatedAt *time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP;onUpdate:CURRENT_TIMESTAMP;not null" json:"-"`
}
