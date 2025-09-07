package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/you/go-jobsity-flights/internal/config"
	"net/http"
	"net/url"
	"time"
)

type RapidBooking struct {
	host        string
	path        string
	rapidApiKey string
	client      *http.Client
}

func NewRapidBooking(cfg *config.Config) *RapidBooking {
	return &RapidBooking{host: cfg.RapidBookingHost,
		path:        "/api/v1/flights/searchFlights",
		rapidApiKey: cfg.RapidBookingRapidApiKey,
		client:      http.DefaultClient,
	}
}

func (r *RapidBooking) Name() string {
	return "rapid-booking"
}

func (r *RapidBooking) Search(ctx context.Context, origin, destination, date string) ([]FlightOffer, error) {
	if r.rapidApiKey == "" {
		return nil, fmt.Errorf("rapid booking: missing API key")
	}

	u := url.URL{
		Scheme: "https",
		Host:   r.host,
		Path:   r.path,
	}
	q := u.Query()
	// Rapid requires the “.AIRPORT” suffix
	q.Set("fromId", origin+".AIRPORT")
	q.Set("toId", destination+".AIRPORT")
	q.Set("departDate", date) // YYYY-MM-DD
	q.Set("stops", "none")    // only direct to keep parsing simple/fastest
	q.Set("pageNo", "1")
	q.Set("adults", "1")
	q.Set("children", "0") // change if you want kids/infants
	q.Set("sort", "BEST")
	q.Set("cabinClass", "ECONOMY")
	q.Set("currency_code", "EUR")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	req.Header.Set("X-RapidAPI-Key", r.rapidApiKey)
	req.Header.Set("X-RapidAPI-Host", r.host)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rapid booking: %s", resp.Status)
	}

	var payload struct {
		Data struct {
			FlightOffers []struct {
				Segments []struct {
					DepartureTime string `json:"departureTime"`
					ArrivalTime   string `json:"arrivalTime"`
					TotalTime     int    `json:"totalTime"`
				} `json:"segments"`
				PriceBreakdown struct {
					Total struct {
						CurrencyCode string `json:"currencyCode"`
						Units        int64  `json:"units"`
						Nanos        int64  `json:"nanos"`
					} `json:"total"`
				} `json:"priceBreakdown"`
			} `json:"flightOffers"`
		} `json:"data"`
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if !payload.Status {
		return nil, fmt.Errorf("rapid booking: %s", payload.Message)
	}

	fmt.Println(payload)

	var out []FlightOffer
	for _, fo := range payload.Data.FlightOffers {
		if len(fo.Segments) == 0 {
			continue
		}
		seg := fo.Segments[0]

		dep := parseRapidTime(seg.DepartureTime)
		arr := parseRapidTime(seg.ArrivalTime)

		durMin := seg.TotalTime / 60
		if durMin <= 0 && !dep.IsZero() && !arr.IsZero() {
			durMin = int(arr.Sub(dep).Minutes())
		}

		total := float64(fo.PriceBreakdown.Total.Units) +
			float64(fo.PriceBreakdown.Total.Nanos)/1e9

		out = append(out, FlightOffer{
			Provider:    r.Name(),
			Price:       total,
			Currency:    fo.PriceBreakdown.Total.CurrencyCode,
			DurationMin: durMin,
			DepartAt:    dep,
			ArriveAt:    arr,
		})
	}

	return out, nil
}

func parseRapidTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}
