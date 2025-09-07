package service

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/you/go-jobsity-flights/internal/config"
	"github.com/you/go-jobsity-flights/internal/providers"
)

func TestSearchCheapestFastest(t *testing.T) {

	cfg := &config.Config{
		SearchTimeout: 5 * time.Second,
		CacheTTL:      5 * time.Second,
	}

	p1 := ProviderMock{name: "p1",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p1",
				Price:       200,
				Currency:    "EUR",
				DurationMin: 120,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		},
	}
	p2 := ProviderMock{name: "p2",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p2",
				Price:       150,
				Currency:    "EUR",
				DurationMin: 90,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}
	p3 := ProviderMock{name: "p3",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p3",
				Price:       300,
				Currency:    "EUR",
				DurationMin: 60,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}

	svc := NewSearchService([]providers.FlightProvider{p1, p2, p3},
		cfg.SearchTimeout,
		cfg.CacheTTL)
	res, err := svc.Search(context.Background(), "AMS", "BCN", "2025-10-01")
	if err != nil {
		t.Fatal(err)
	}
	if res.Cheapest.Provider != "p2" {
		t.Fatalf("expected cheapest from p2, got %s", res.Cheapest.Provider)
	}
	if res.Fastest.Provider != "p3" {
		t.Fatalf("expected fastest from p3, got %s", res.Fastest.Provider)
	}
	if len(res.All) != 3 {
		t.Fatalf("expected 3 offers, got %d", len(res.All))
	}
}

func valToPtr[T any](param T) *T {
	return &param
}

func TestSearchError(t *testing.T) {

	cfg := &config.Config{
		SearchTimeout: 5 * time.Second,
		CacheTTL:      5 * time.Second,
	}

	p1 := ProviderMock{name: "p1",
		cfg:             cfg,
		errorOutMessage: valToPtr("API Request Fail"),
	}
	p2 := ProviderMock{name: "p2",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p2",
				Price:       150,
				Currency:    "EUR",
				DurationMin: 90,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}
	p3 := ProviderMock{name: "p3",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p3",
				Price:       300,
				Currency:    "EUR",
				DurationMin: 60,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}

	svc := NewSearchService([]providers.FlightProvider{p1, p2, p3},
		cfg.SearchTimeout,
		cfg.CacheTTL)
	_, err := svc.Search(context.Background(), "AMS", "BCN", "2025-10-01")
	require.Error(t, err)
	require.Equal(t, "p1: API Request Fail", err.Error())
}

func TestSearchTimeOut(t *testing.T) {

	cfg := &config.Config{
		SearchTimeout: 1 * time.Second,
		CacheTTL:      5 * time.Second,
	}

	p1 := ProviderMock{name: "p1",
		cfg:   cfg,
		delay: 2 * time.Second,
	}
	p2 := ProviderMock{name: "p2",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p2",
				Price:       150,
				Currency:    "EUR",
				DurationMin: 90,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}
	p3 := ProviderMock{name: "p3",
		cfg: cfg,
		offers: []providers.FlightOffer{
			{Provider: "p3",
				Price:       300,
				Currency:    "EUR",
				DurationMin: 60,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		}}

	svc := NewSearchService([]providers.FlightProvider{p1, p2, p3},
		cfg.SearchTimeout,
		cfg.CacheTTL)
	_, err := svc.Search(context.Background(), "AMS", "BCN", "2025-10-01")
	require.Error(t, err)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestSearch_NoOffers(t *testing.T) {
	cfg := &config.Config{
		SearchTimeout: 5 * time.Second,
		CacheTTL:      5 * time.Second,
	}

	empty1 := &ProviderMock{name: "p1",
		cfg:    cfg,
		offers: []providers.FlightOffer{}}
	empty2 := &ProviderMock{name: "p2",
		cfg:    cfg,
		offers: []providers.FlightOffer{}}

	s := NewSearchService([]providers.FlightProvider{empty1, empty2},
		cfg.SearchTimeout,
		cfg.CacheTTL)

	_, err := s.Search(context.Background(), "AAA", "BBB", "2025-01-01")
	if err == nil {
		t.Fatalf("expected 'no offers found' error, got nil")
	}
	if err.Error() != "no offers found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_CacheHit(t *testing.T) {

	cfg := &config.Config{
		SearchTimeout: 5 * time.Second,
		CacheTTL:      1 * time.Second,
	}

	var calls int32

	prov := &ProviderMock{name: "p1",
		cfg:       cfg,
		callCount: &calls,
		offers: []providers.FlightOffer{
			{Provider: "p2",
				Price:       150,
				Currency:    "EUR",
				DurationMin: 90,
				DepartAt:    time.Now(),
				ArriveAt:    time.Now()},
		},
	}

	s := NewSearchService([]providers.FlightProvider{prov},
		cfg.SearchTimeout,
		cfg.CacheTTL)

	ctx := context.Background()
	res1, err := s.Search(ctx, "GRU", "JFK", "2025-09-15")
	if err != nil {
		t.Fatalf("first Search error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls after first search: got %d, want 1", got)
	}

	{
		// Same key -> should hit cache, not call provider again
		res2, err := s.Search(ctx, "GRU", "JFK", "2025-09-15")
		if err != nil {
			t.Fatalf("second Search (cache) error: %v", err)
		}
		if got := atomic.LoadInt32(&calls); got != 1 {
			t.Fatalf("provider should not have been called on cache hit; calls=%d", got)
		}
		// Results should be identical
		if !reflect.DeepEqual(res1, res2) {
			t.Fatalf("cached result differs from original\nres1=%+v\nres2=%+v", res1, res2)
		}
	}

	time.Sleep(2 * time.Second)

	{
		// Same key -> should hit cache, not call provider again
		res2, err := s.Search(ctx, "GRU", "JFK", "2025-09-15")
		if err != nil {
			t.Fatalf("second Search (cache) error: %v", err)
		}
		if got := atomic.LoadInt32(&calls); got != 2 {
			t.Fatalf("provider should have been called after 2 seconds; calls=%d", got)
		}
		// Results should be identical
		if !reflect.DeepEqual(res1, res2) {
			t.Fatalf("cached result differs from original\nres1=%+v\nres2=%+v", res1, res2)
		}
	}

}
