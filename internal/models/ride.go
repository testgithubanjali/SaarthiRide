package models

import "time"

type Ride struct {
	ID          uint  `gorm:"primaryKey" json:"id"`
	PassengerID uint  `gorm:"not null;index" json:"passenger_id"`
	DriverID    *uint `gorm:"index" json:"driver_id"`
	VehicleID   *uint `gorm:"index" json:"vehicle_id"`

	PickupLatitude  float64 `gorm:"not null" json:"pickup_latitude"`
	PickupLongitude float64 `gorm:"not null" json:"pickup_longitude"`

	DestinationLatitude  float64 `gorm:"not null" json:"destination_latitude"`
	DestinationLongitude float64 `gorm:"not null" json:"destination_longitude"`

	Status string `gorm:"not null;default:pending;index" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
