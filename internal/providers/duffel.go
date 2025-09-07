package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/you/go-jobsity-flights/internal/config"
)

type Duffel struct {
	host   string
	token  string
	client *http.Client
}

func NewDuffel(cfg *config.Config) *Duffel {
	return &Duffel{host: cfg.DuffelHost,
		token:  cfg.DuffelToken,
		client: http.DefaultClient,
	}
}

func (d *Duffel) Name() string {
	return "duffel"
}

type duffelOfferRequest struct {
	Slices []struct {
		Origin        string `json:"origin"`
		Destination   string `json:"destination"`
		DepartureDate string `json:"departure_date"`
	} `json:"slices"`
	Passengers []struct {
		Type string `json:"type"`
	} `json:"passengers"`
	CabinClass   string `json:"cabin_class"`
	CurrencyCode string `json:"currency"`
	ReturnOffers bool   `json:"return_offers"`
}

type duffelOfferRequestEnvelope struct {
	Data duffelOfferRequest `json:"data"`
}

type duffelOffer struct {
	TotalAmount   string `json:"total_amount"`
	TotalCurrency string `json:"total_currency"`
	Slices        []struct {
		Segments []struct {
			DepartingAt string `json:"departing_at"`
			ArrivingAt  string `json:"arriving_at"`
			Duration    string `json:"duration"` // ISO8601 e.g. PT2H10M
		} `json:"segments"`
	} `json:"slices"`
}

type duffelOfferResp struct {
	Data struct {
		Offers []duffelOffer `json:"offers"`
	} `json:"data"`
}

func (d *Duffel) Search(ctx context.Context, origin, destination, date string) ([]FlightOffer, error) {
	if d.token == "" {
		return nil, errors.New("duffel token missing")
	}

	reqBody := duffelOfferRequestEnvelope{Data: duffelOfferRequest{
		Slices: []struct {
			Origin        string `json:"origin"`
			Destination   string `json:"destination"`
			DepartureDate string `json:"departure_date"`
		}([]struct {
			Origin, Destination, DepartureDate string `json:"origin" json:"destination" json:"departure_date"`
		}{
			{Origin: origin, Destination: destination, DepartureDate: date},
		}),
		Passengers: []struct {
			Type string `json:"type"`
		}{{Type: "adult"}},
		CabinClass:   "economy",
		CurrencyCode: "EUR",
		ReturnOffers: true,
	}}
	b, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, d.host+"/air/offer_requests", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Duffel-Version", "v2")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("duffel: %s", resp.Status)
	}

	var pr duffelOfferResp
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}

	var out []FlightOffer
	for _, o := range pr.Data.Offers {
		if len(o.Slices) == 0 || len(o.Slices[0].Segments) == 0 {
			continue
		}
		seg0 := o.Slices[0].Segments[0]
		segn := o.Slices[0].Segments[len(o.Slices[0].Segments)-1]
		depart := mustParseDuffelTime(seg0.DepartingAt)
		arrive := mustParseDuffelTime(segn.ArrivingAt)
		dur := parseISODurationMinutes(seg0.Duration)
		price, _ := strconv.ParseFloat(o.TotalAmount, 64)
		currency := "EUR"
		out = append(out, FlightOffer{Provider: d.Name(),
			Price:       price,
			Currency:    currency,
			DurationMin: dur,
			DepartAt:    depart,
			ArriveAt:    arrive})
	}
	return out, nil
}

func mustParseDuffelTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}
