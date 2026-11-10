package trip

import (
	"github.com/jackc/pgx/v5"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

func scanTrip(row pgx.Row) (*domain.Trip, error) {
	var trip domain.Trip

	err := row.Scan(
		&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.Start.Latitude, &trip.Start.Longitude,
		&trip.End.Latitude, &trip.End.Longitude,
		&trip.Price, &trip.Status,
		&trip.StartedAt, &trip.FinishedAt,
		&trip.CreatedAt, &trip.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}
