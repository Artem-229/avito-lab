package create

import (
	"fmt"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/domainerror"
)

func validateTripData(req api.TripData) error {
	trip := domain.Trip{
		UserID: req.UserId, DriverID: req.DriverId, Price: req.Price,
		Start: domain.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		End:   domain.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
	}
	if err := trip.Validate(); err != nil {
		return fmt.Errorf("%w: %w", domainerror.ErrInvalidRequest, err)
	}
	return nil
}
