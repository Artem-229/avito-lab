package trip

import (
	"fmt"
	"math"

	"github.com/google/uuid"
)

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

func (t *Trip) Validate() error {
	if t.UserID == uuid.Nil {
		return &ValidationError{"user_id", "is required"}
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{"driver_id", "is required"}
	}
	switch t.Status {
	case "":
	case StatusActive:
		if t.FinishedAt != nil {
			return &ValidationError{"finished_at", "must be empty for active trip"}
		}
	case StatusCompleted:
		if t.FinishedAt == nil {
			return &ValidationError{"finished_at", "is required for completed trip"}
		}
	default:
		return &ValidationError{"status", "must be active or completed"}
	}
	if t.Price < 0 {
		return &ValidationError{"price", "must be >= 0"}
	}
	for _, point := range []struct {
		name  string
		value Point
	}{{"start_point", t.Start}, {"end_point", t.End}} {
		if math.IsNaN(point.value.Latitude) || math.IsInf(point.value.Latitude, 0) || point.value.Latitude < -90 || point.value.Latitude > 90 {
			return &ValidationError{point.name + ".latitude", "must be in [-90, 90]"}
		}
		if math.IsNaN(point.value.Longitude) || math.IsInf(point.value.Longitude, 0) || point.value.Longitude < -180 || point.value.Longitude > 180 {
			return &ValidationError{point.name + ".longitude", "must be in [-180, 180]"}
		}
	}
	return nil
}
