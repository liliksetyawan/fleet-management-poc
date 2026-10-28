package service

import (
	"context"
	"fleet-management-poc/internal/domain"
	"math"
	"sync"
)

type LocationStore interface {
	Save(context.Context, domain.Location) error
}
type EventPublisher interface {
	Publish(context.Context, domain.GeofenceEvent) error
}
type FleetService struct {
	store                      LocationStore
	publisher                  EventPublisher
	fenceLat, fenceLon, radius float64
	mu                         sync.Mutex
	inside                     map[string]bool
}

func NewFleetService(s LocationStore, p EventPublisher, lat, lon, radius float64) *FleetService {
	return &FleetService{store: s, publisher: p, fenceLat: lat, fenceLon: lon, radius: radius, inside: map[string]bool{}}
}
func (s *FleetService) Process(ctx context.Context, l domain.Location) error {
	if err := l.Validate(); err != nil {
		return err
	}
	if err := s.store.Save(ctx, l); err != nil {
		return err
	}
	inside := distanceMeters(l.Latitude, l.Longitude, s.fenceLat, s.fenceLon) <= s.radius
	s.mu.Lock()
	wasInside, known := s.inside[l.VehicleID]
	s.inside[l.VehicleID] = inside
	s.mu.Unlock()
	if inside && (!known || !wasInside) && s.publisher != nil {
		e := domain.GeofenceEvent{VehicleID: l.VehicleID, Event: "geofence_entry", Timestamp: l.Timestamp}
		e.Location.Latitude = l.Latitude
		e.Location.Longitude = l.Longitude
		return s.publisher.Publish(ctx, e)
	}
	return nil
}
func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
