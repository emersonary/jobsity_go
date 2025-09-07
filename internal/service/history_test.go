package service

import (
	"reflect"
	"testing"
	"time"
)

// Helper to check monotonic month order with YYYY-MM format.
// Since it's YYYY-MM lexicographically increasing == chronologically increasing.
func isMonotonicMonths(points []MonthPoint) bool {
	for i := 1; i < len(points); i++ {
		if points[i-1].Month > points[i].Month {
			return false
		}
	}
	return true
}

func TestMonthlyAverages_LengthAndDefaults(t *testing.T) {
	h := NewHistoryService()

	out := h.MonthlyAverages("GRU", "JFK", 3)
	if got, want := len(out), 3; got != want {
		t.Fatalf("length: got %d, want %d", got, want)
	}

	out2 := h.MonthlyAverages("GRU", "JFK", 0) // defaults to 24
	if got, want := len(out2), 24; got != want {
		t.Fatalf("default length: got %d, want %d", got, want)
	}

	out3 := h.MonthlyAverages("GRU", "JFK", -5) // also defaults to 24
	if got, want := len(out3), 24; got != want {
		t.Fatalf("negative months -> default length: got %d, want %d", got, want)
	}
}

func TestMonthlyAverages_OrderFormatCurrency(t *testing.T) {
	h := NewHistoryService()
	const months = 6

	out := h.MonthlyAverages("GRU", "JFK", months)

	// Order: oldest -> newest
	if !isMonotonicMonths(out) {
		t.Fatalf("months are not in monotonically increasing (oldest->newest) order")
	}

	// Last month should be current UTC month (format YYYY-MM)
	nowMonth := time.Now().UTC().Format("2006-01")
	if got := out[len(out)-1].Month; got != nowMonth {
		t.Fatalf("last month: got %q, want %q", got, nowMonth)
	}

	// All currencies should be EUR
	for i, mp := range out {
		if mp.Currency != "EUR" {
			t.Fatalf("currency at idx %d: got %q, want %q", i, mp.Currency, "EUR")
		}
	}
}

func TestMonthlyAverages_DeterministicValues(t *testing.T) {
	h := NewHistoryService()
	origin, dest := "ABC", "XYZ"
	const months = 7

	out := h.MonthlyAverages(origin, dest, months)

	// Recompute expected values from the same formula the service uses.
	base := 120.0
	salt := float64(len(origin)*13 + len(dest)*7)

	for idx, mp := range out {
		i := months - 1 - idx // matches the generator's "i" for this row

		// Derive season from the returned month string.
		mt, err := time.Parse("2006-01", mp.Month)
		if err != nil {
			t.Fatalf("bad month format at idx %d: %q: %v", idx, mp.Month, err)
		}
		season := 1.0
		switch mt.Month() {
		case time.July, time.August, time.December:
			season = 1.25
		}

		expected := base*season + float64((i%5)*6) + salt
		expected = round2(expected) // use the same rounding as production

		if mp.AvgPrice != expected {
			t.Fatalf("price at idx %d (%s): got %.2f, want %.2f", idx, mp.Month, mp.AvgPrice, expected)
		}
	}
}

func TestMonthlyAverages_DeterministicAcrossCalls(t *testing.T) {
	h := NewHistoryService()
	origin, dest := "GRU", "JFK"
	const months = 9

	out1 := h.MonthlyAverages(origin, dest, months)
	out2 := h.MonthlyAverages(origin, dest, months)

	if !reflect.DeepEqual(out1, out2) {
		t.Fatalf("results differ across calls with same inputs\nout1=%v\nout2=%v", out1, out2)
	}
}

func TestMonthlyAverages_SingleMonth(t *testing.T) {
	h := NewHistoryService()
	out := h.MonthlyAverages("AAA", "BBB", 1)
	if len(out) != 1 {
		t.Fatalf("got %d points, want 1", len(out))
	}
	// The only month should be the current UTC month
	nowMonth := time.Now().UTC().Format("2006-01")
	if out[0].Month != nowMonth {
		t.Fatalf("month: got %q, want %q", out[0].Month, nowMonth)
	}
}
