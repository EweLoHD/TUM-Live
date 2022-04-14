package model

import (
	"gorm.io/gorm"
)

type VideoSection struct {
	gorm.Model

	Description    string `gorm:"not null" json:"description"`
	StartHours uint   `gorm:"not null" json:"startHours"`
	StartMinutes uint   `gorm:"not null" json:"startMinutes"`
	StartSeconds uint   `gorm:"not null" json:"startSeconds"`

	StreamID       uint   `gorm:"not null" json:"streamID"`
}