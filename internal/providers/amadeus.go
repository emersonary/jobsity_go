package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/you/go-jobsity-flights/internal/config"
)

type Amadeus struct {
	host       string
	authPath   string
	searchPath string
	client     *http.Client
	id         string
	secret     string
	mu         sync.Mutex
	tok        string
	expires    time.Time
}

func NewAmadeus(cfg *config.Config) *Amadeus {
	return &Amadeus{host: cfg.AmadeusURL,
		authPath:   "/v1/security/oauth2/token",
		searchPath: "/v2/shopping/flight-offers",
		id:         cfg.AmadeusClientId,
		secret:     cfg.AmadeusClientSSecret,
		client:     http.DefaultClient,
	}
}

func (a *Amadeus) Name() string { return "amadeus" }

func (a *Amadeus) token(ctx context.Context) (string, error) {
	//a.mu.Lock()
	//defer a.mu.Unlock()
	//if a.tok != "" && time.Now().Before(a.expires.Add(-10*time.Second)) {
	//	return a.tok, nil
	//}
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", a.id)
	data.Set("client_secret", a.secret)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.host+a.authPath, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("amadeus token: %s", resp.Status)
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	a.tok = tr.AccessToken
	a.expires = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return a.tok, nil
}

func (a *Amadeus) Search(ctx context.Context, origin, destination, date string) ([]FlightOffer, error) {
	if a.id == "" || a.secret == "" {
		return nil, errors.New("amadeus credentials missing")
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s%s?originLocationCode=%s&destinationLocationCode=%s&departureDate=%s&adults=1&currencyCode=EUR&max=5",
		a.host,
		a.searchPath,
		url.QueryEscape(origin),
		url.QueryEscape(destination),
		url.QueryEscape(date))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("amadeus search: %s - %s - %s", resp.Status, u, tok)
	}

	var payload struct {
		Data []struct {
			Price struct {
				Total string `json:"total"`
			} `json:"price"`
			Itineraries []struct {
				Duration string `json:"duration"` // ISO8601 e.g. PT2H10M
				Segments []struct {
					Departure struct {
						At string `json:"at"`
					}
					Arrival struct {
						At string `json:"at"`
					}
				} `json:"segments"`
			} `json:"itineraries"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var out []FlightOffer
	for _, d := range payload.Data {
		if len(d.Itineraries) == 0 || len(d.Itineraries[0].Segments) == 0 {
			continue
		}
		price, _ := strconv.ParseFloat(d.Price.Total, 64)
		segFirst := d.Itineraries[0].Segments[0]
		segLast := d.Itineraries[0].Segments[len(d.Itineraries[0].Segments)-1]
		depart, _ := parseAmadeusTime(segFirst.Departure.At)
		arrive, _ := parseAmadeusTime(segLast.Arrival.At)
		dur := parseISODurationMinutes(d.Itineraries[0].Duration)
		out = append(out, FlightOffer{
			Provider:    a.Name(),
			Price:       price,
			Currency:    "EUR",
			DurationMin: dur,
			DepartAt:    depart,
			ArriveAt:    arrive,
		})
	}
	return out, nil
}

func parseISODurationMinutes(s string) int {
	// very small parser for formats like PT2H10M, PT150M
	s = strings.TrimPrefix(s, "PT")
	total := 0
	var num strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			num.WriteRune(r)
			continue
		}
		v, _ := strconv.Atoi(num.String())
		num.Reset()
		switch r {
		case 'H':
			total += v * 60
		case 'M':
			total += v
		}
	}
	return total
}

func parseAmadeusTime(s string) (time.Time, error) {
	// Amadeus returns local time without offset, e.g. 2025-09-10T08:45:00
	// Treat it as "naive" time. Use Local (or time.UTC if you prefer).
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	// Fallbacks if they ever include zone
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %q", s)
}
