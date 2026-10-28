package domain

import "errors"

var ErrInvalidLocation = errors.New("invalid vehicle location")

type Location struct {
	VehicleID string  `json:"vehicle_id" db:"vehicle_id"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
	Timestamp int64   `json:"timestamp" db:"timestamp"`
}

type GeofenceEvent struct {
	VehicleID string `json:"vehicle_id"`
	Event     string `json:"event"`
	Location  struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Timestamp int64 `json:"timestamp"`
}

func (l Location) Validate() error {
	if l.VehicleID == "" || len(l.VehicleID) > 64 || l.Latitude < -90 || l.Latitude > 90 || l.Longitude < -180 || l.Longitude > 180 || l.Timestamp <= 0 {
		return ErrInvalidLocation
	}
	return nil
}
