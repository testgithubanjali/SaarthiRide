package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Phone        string    `gorm:"uniqueIndex;not null" json:"phone"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"not null;default:passenger" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}
