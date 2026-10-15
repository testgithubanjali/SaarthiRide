package models

import "time"

type Vehicle struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	DriverID           uint      `gorm:"not null;index" json:"driver_id"`
	RegistrationNumber string    `gorm:"uniqueIndex;not null" json:"registration_number"`
	Model              string    `gorm:"not null" json:"model"`
	Capacity           int       `gorm:"not null;default:3" json:"capacity"`
	IsAvailable        bool      `gorm:"not null;default:false" json:"is_available"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
