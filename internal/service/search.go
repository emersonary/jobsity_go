package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/you/go-jobsity-flights/internal/providers"
	"golang.org/x/sync/errgroup"
)

type SearchResult struct {
	Cheapest providers.FlightOffer   `json:"cheapest"`
	Fastest  providers.FlightOffer   `json:"fastest"`
	All      []providers.FlightOffer `json:"all"`
}

type cacheEntry struct {
	value     SearchResult
	expiresAt time.Time
}

type SearchService struct {
	providers     []providers.FlightProvider
	cache         map[string]cacheEntry
	mu            sync.RWMutex
	searchTimeout time.Duration
	cacheTTL      time.Duration
}

func NewSearchService(prov []providers.FlightProvider, timeout, ttl time.Duration) *SearchService {
	return &SearchService{
		providers:     prov,
		cache:         make(map[string]cacheEntry),
		searchTimeout: timeout,
		cacheTTL:      ttl,
	}
}

func (s *SearchService) cacheKey(origin, dest, date string) string {
	return origin + "|" + dest + "|" + date
}

func (s *SearchService) Search(ctx context.Context, origin, dest, date string) (SearchResult, error) {
	key := s.cacheKey(origin, dest, date)
	// fast cache path
	s.mu.RLock()
	if ce, ok := s.cache[key]; ok && time.Now().Before(ce.expiresAt) {
		s.mu.RUnlock()
		return ce.value, nil
	}
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, s.searchTimeout)
	defer cancel()

	var mu sync.Mutex
	var all []providers.FlightOffer
	g, ctx := errgroup.WithContext(ctx)

	for _, p := range s.providers {
		p := p
		g.Go(func() error {
			offers, err := p.Search(ctx, origin, dest, date)
			if err != nil {
				return err
			}
			fos := make([]providers.FlightOffer, 0, len(offers))
			for _, o := range offers {
				fos = append(fos, providers.FlightOffer{
					Provider:    o.Provider,
					Price:       o.Price,
					Currency:    o.Currency,
					DurationMin: o.DurationMin,
					DepartAt:    o.DepartAt,
					ArriveAt:    o.ArriveAt,
				})
			}
			mu.Lock()
			all = append(all, fos...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return SearchResult{}, err
	}
	if len(all) == 0 {
		return SearchResult{}, errors.New("no offers found")
	}

	sortedByPrice := append([]providers.FlightOffer(nil), all...)
	sort.Slice(sortedByPrice, func(i, j int) bool { return sortedByPrice[i].Price < sortedByPrice[j].Price })
	cheapest := sortedByPrice[0]

	sortedByDuration := append([]providers.FlightOffer(nil), all...)
	sort.Slice(sortedByDuration, func(i, j int) bool { return sortedByDuration[i].DurationMin < sortedByDuration[j].DurationMin })
	fastest := sortedByDuration[0]

	sort.Slice(all, func(i, j int) bool {
		if all[i].Price != all[j].Price {
			return all[i].Price < all[j].Price
		}
		if all[i].DurationMin != all[j].DurationMin {
			return all[i].DurationMin < all[j].DurationMin
		}
		return all[i].DepartAt.Before(all[j].DepartAt)
	})

	res := SearchResult{Cheapest: cheapest, Fastest: fastest, All: all}

	s.mu.Lock()
	s.cache[key] = cacheEntry{value: res, expiresAt: time.Now().Add(s.cacheTTL)}
	s.mu.Unlock()

	return res, nil
}
