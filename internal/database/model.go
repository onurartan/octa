package database

import (
	"time"
)

type Image struct {
	ID   string `gorm:"primaryKey" json:"id"`
	Data []byte `gorm:"type:blob" json:"-"` // Image

	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"` // "jpeg", "png", "webp"
	Size   int64  `json:"size"`

	Mappings  []KeyMapping `gorm:"foreignKey:ImageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UpdatedAt time.Time    `gorm:"autoUpdateTime"`
	CreatedAt time.Time    `json:"created_at"`
}

type KeyMapping struct {
	Key       string    `gorm:"primaryKey;type:text"` // runo, email@...
	ImageID   string    `gorm:"index;type:text"`
	CreatedAt time.Time `json:"created_at"`
}
