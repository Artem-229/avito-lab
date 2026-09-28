package entities

import "errors"

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrDriverBusy    = errors.New("driver is busy")
	ErrTripCompleted = errors.New("trip completed")
)
