package service

import "time"

type MonthPoint struct {
	Month    string  `json:"month"` // YYYY-MM
	AvgPrice float64 `json:"avg_price"`
	Currency string  `json:"currency"`
}

// HistoryService returns a synthetic, deterministic 24-month series based on route.
// In real life, back this with a DB and ingest from providers.
type HistoryService struct{}

func NewHistoryService() *HistoryService {
	return &HistoryService{}
}

func (h *HistoryService) MonthlyAverages(origin, dest string, months int) []MonthPoint {
	if months <= 0 {
		months = 24
	}
	base := 120.0
	// simple deterministic variance based on rune sums
	salt := float64(len(origin)*13 + len(dest)*7)
	now := time.Now().UTC()
	out := make([]MonthPoint, 0, months)
	for i := months - 1; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		season := 1.0
		if m.Month() == time.July || m.Month() == time.August || m.Month() == time.December {
			season = 1.25
		}
		price := base*season + float64((i%5)*6) + salt
		out = append(out, MonthPoint{Month: m.Format("2006-01"), AvgPrice: round2(price), Currency: "EUR"})
	}
	return out
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
