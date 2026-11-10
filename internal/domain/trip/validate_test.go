package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Artem-229/avito-lab/internal/domain/trip"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		edit        func(*trip.Trip)
	}{
		{"valid", "", func(*trip.Trip) {}},
		{"missing user", "user_id", func(v *trip.Trip) { v.UserID = uuid.Nil }},
		{"missing driver", "driver_id", func(v *trip.Trip) { v.DriverID = uuid.Nil }},
		{"negative price", "price", func(v *trip.Trip) { v.Price = -1 }},
		{"low latitude", "start_point.latitude", func(v *trip.Trip) { v.Start.Latitude = -90.01 }},
		{"high latitude", "end_point.latitude", func(v *trip.Trip) { v.End.Latitude = 90.01 }},
		{"low longitude", "start_point.longitude", func(v *trip.Trip) { v.Start.Longitude = -180.01 }},
		{"high longitude", "end_point.longitude", func(v *trip.Trip) { v.End.Longitude = 180.01 }},
		{"nan", "start_point.latitude", func(v *trip.Trip) { v.Start.Latitude = math.NaN() }},
		{"infinity", "end_point.longitude", func(v *trip.Trip) { v.End.Longitude = math.Inf(1) }},
		{"active with finish", "finished_at", func(v *trip.Trip) { now := time.Now(); v.FinishedAt = &now }},
		{"completed without finish", "finished_at", func(v *trip.Trip) { v.Status = trip.StatusCompleted }},
		{"unknown status", "status", func(v *trip.Trip) { v.Status = "unknown" }},
		{"boundaries", "", func(v *trip.Trip) {
			v.Start = trip.Point{Latitude: -90, Longitude: -180}
			v.End = trip.Point{Latitude: 90, Longitude: 180}
		}},
		{"completed", "", func(v *trip.Trip) { now := time.Now(); v.Status = trip.StatusCompleted; v.FinishedAt = &now }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := trip.Trip{UserID: uuid.New(), DriverID: uuid.New(), Status: trip.StatusActive}
			tc.edit(&model)
			err := model.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var validation *trip.ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field || validation.Message == "" {
				t.Fatalf("validation=%v, want field %s", err, tc.field)
			}
		})
	}
}
