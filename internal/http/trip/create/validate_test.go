package create

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/domainerror"
)

func TestValidateTripData(t *testing.T) {
	valid := api.TripData{
		UserId: uuid.New(), DriverId: uuid.New(), Price: 100,
		StartPoint: api.Coordinates{Latitude: 59.9, Longitude: 30.3},
		EndPoint:   api.Coordinates{Latitude: 59.8, Longitude: 30.2},
	}
	tests := []struct {
		name string
		edit func(*api.TripData)
	}{
		{name: "valid"},
		{name: "missing user", edit: func(v *api.TripData) { v.UserId = uuid.Nil }},
		{name: "missing driver", edit: func(v *api.TripData) { v.DriverId = uuid.Nil }},
		{name: "negative price", edit: func(v *api.TripData) { v.Price = -1 }},
		{name: "latitude below range", edit: func(v *api.TripData) { v.StartPoint.Latitude = -90.1 }},
		{name: "latitude above range", edit: func(v *api.TripData) { v.EndPoint.Latitude = 90.1 }},
		{name: "longitude below range", edit: func(v *api.TripData) { v.StartPoint.Longitude = -180.1 }},
		{name: "longitude above range", edit: func(v *api.TripData) { v.EndPoint.Longitude = 180.1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			if tt.edit != nil {
				tt.edit(&input)
			}
			err := validateTripData(input)
			if tt.name == "valid" {
				if err != nil {
					t.Fatalf("validateTripData() error = %v", err)
				}
				return
			}
			if !errors.Is(err, domainerror.ErrInvalidRequest) {
				t.Fatalf("validateTripData() error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}
