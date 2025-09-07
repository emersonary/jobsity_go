package service

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/you/go-jobsity-flights/internal/config"
	"github.com/you/go-jobsity-flights/internal/providers"
)

type ProviderMock struct {
	name            string
	offers          []providers.FlightOffer
	delay           time.Duration
	errorOutMessage *string
	cfg             *config.Config
	callCount       *int32
}

func (p ProviderMock) Name() string {
	return p.name
}

func (p ProviderMock) Search(ctx context.Context, o, d, dt string) ([]providers.FlightOffer, error) {
	if p.callCount != nil {
		atomic.AddInt32(p.callCount, 1)
	}
	if p.errorOutMessage != nil {
		return nil, errors.New(p.Name() + ": " + *p.errorOutMessage)
	}
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return p.offers, nil
}
