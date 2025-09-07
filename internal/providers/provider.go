package providers

import (
	"context"
	"time"
)

type FlightOffer struct {
	Provider    string    `json:"provider"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	DurationMin int       `json:"duration_min"`
	DepartAt    time.Time `json:"depart_at"`
	ArriveAt    time.Time `json:"arrive_at"`
}

type FlightProvider interface {
	Name() string
	Search(ctx context.Context, origin, destination, date string) ([]FlightOffer, error)
}
