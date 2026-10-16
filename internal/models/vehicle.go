package models

import "time"

type Vehicle struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	DriverID           uint      `gorm:"not null;index" json:"driver_id"`
	RegistrationNumber string    `gorm:"uniqueIndex;not null" json:"registration_number"`
	Model              string    `gorm:"not null" json:"model"`
	Capacity           int       `gorm:"not null;default:3" json:"capacity"`
	IsAvailable        bool      `gorm:"not null;default:false" json:"is_available"`
	Latitude           *float64  `gorm:"type:double precision" json:"latitude"`
	Longitude          *float64  `gorm:"type:double precision" json:"longitude"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
