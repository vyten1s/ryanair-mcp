package ryanair_test

import (
	"testing"

	"github.com/adambenhassen/ryanair-mcp/internal/ryanair"
)

func TestCheapestWeekendTripMatcher(t *testing.T) {
	price := func(v float64) *float64 { return &v }
	fare := func(day string, p *float64) ryanair.DailyFare {
		return ryanair.DailyFare{Day: day, Price: p, Currency: "EUR"}
	}
	cases := []struct {
		name                 string
		outbounds            []ryanair.DailyFare
		inbounds             []ryanair.DailyFare
		minNights, maxNights int
		wantNil              bool
		wantOut, wantIn      string
		wantTotal            float64
	}{
		{
			name: "picks cheapest pair across Fri/Sat/Sun outbounds",
			outbounds: []ryanair.DailyFare{
				fare("2026-07-03", price(20)), // Fri
				fare("2026-07-04", price(10)), // Sat (cheapest valid)
				fare("2026-07-02", price(1)),  // Thu - must be ignored despite lowest price
			},
			inbounds: []ryanair.DailyFare{
				fare("2026-07-05", price(25)), // Sun
				fare("2026-07-06", price(7)),  // Mon
			},
			minNights: 1, maxNights: 3, wantOut: "2026-07-04", wantIn: "2026-07-06", wantTotal: 17,
		},
		{
			name:      "Sun->Mon one-night trip is allowed",
			outbounds: []ryanair.DailyFare{fare("2026-07-05", price(10))}, // Sun
			inbounds:  []ryanair.DailyFare{fare("2026-07-06", price(5))},  // Mon
			minNights: 1, maxNights: 3, wantOut: "2026-07-05", wantIn: "2026-07-06", wantTotal: 15,
		},
		{
			name:      "Fri->Mon three-night trip is allowed",
			outbounds: []ryanair.DailyFare{fare("2026-07-03", price(10))}, // Fri
			inbounds:  []ryanair.DailyFare{fare("2026-07-06", price(5))},  // Mon (+3)
			minNights: 1, maxNights: 3, wantOut: "2026-07-03", wantIn: "2026-07-06", wantTotal: 15,
		},
		{
			name:      "weekend crossing a month boundary",
			outbounds: []ryanair.DailyFare{fare("2026-07-31", price(12))}, // Fri
			inbounds:  []ryanair.DailyFare{fare("2026-08-02", price(8))},  // Sun
			minNights: 1, maxNights: 3, wantOut: "2026-07-31", wantIn: "2026-08-02", wantTotal: 20,
		},
		{
			name:      "Thursday outbound -> nil",
			outbounds: []ryanair.DailyFare{fare("2026-07-02", price(1))}, // Thu
			inbounds:  []ryanair.DailyFare{fare("2026-07-04", price(1))}, // Sat (+2)
			minNights: 1, maxNights: 3, wantNil: true,
		},
		{
			name:      "Tuesday inbound -> nil",
			outbounds: []ryanair.DailyFare{fare("2026-07-05", price(1))}, // Sun
			inbounds:  []ryanair.DailyFare{fare("2026-07-07", price(1))}, // Tue (+2)
			minNights: 1, maxNights: 3, wantNil: true,
		},
		{
			name:      "respects max nights",
			outbounds: []ryanair.DailyFare{fare("2026-07-03", price(5))}, // Fri
			inbounds:  []ryanair.DailyFare{fare("2026-07-06", price(5))}, // Mon (+3)
			minNights: 1, maxNights: 2, wantNil: true,
		},
		{
			name:      "respects min nights",
			outbounds: []ryanair.DailyFare{fare("2026-07-05", price(5))}, // Sun
			inbounds:  []ryanair.DailyFare{fare("2026-07-06", price(5))}, // Mon (+1)
			minNights: 2, maxNights: 3, wantNil: true,
		},
		{
			name: "skips nil-priced outbound in favor of a priced one",
			outbounds: []ryanair.DailyFare{
				fare("2026-07-10", nil),       // Fri, no price -> skipped
				fare("2026-07-03", price(40)), // Fri, priced
			},
			inbounds: []ryanair.DailyFare{
				fare("2026-07-05", price(10)), // Sun, pairs 07-03
				fare("2026-07-12", price(1)),  // would pair 07-10, but that outbound is unpriced
			},
			minNights: 1, maxNights: 3, wantOut: "2026-07-03", wantIn: "2026-07-05", wantTotal: 50,
		},
		{
			name: "skips inbound that is nil-priced",
			outbounds: []ryanair.DailyFare{
				fare("2026-07-10", price(1)),  // Fri, but its only inbound is unpriced
				fare("2026-07-03", price(40)), // Fri, fully priced pair
			},
			inbounds: []ryanair.DailyFare{
				fare("2026-07-12", nil),       // Sun, unpriced -> skipped
				fare("2026-07-05", price(10)), // Sun, pairs 07-03
			},
			minNights: 1, maxNights: 3, wantOut: "2026-07-03", wantIn: "2026-07-05", wantTotal: 50,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trip, err := ryanair.CheapestWeekendTrip(tc.outbounds, tc.inbounds, tc.minNights, tc.maxNights)
			if err != nil {
				t.Fatalf("matcher: %v", err)
			}
			if tc.wantNil {
				if trip != nil {
					t.Fatalf("want nil trip, got %+v", trip)
				}
				return
			}
			if trip == nil {
				t.Fatal("want a trip, got nil")
			}
			if trip.Outbound.Day != tc.wantOut || trip.Inbound.Day != tc.wantIn {
				t.Errorf("pair = %s->%s, want %s->%s", trip.Outbound.Day, trip.Inbound.Day, tc.wantOut, tc.wantIn)
			}
			if trip.TotalPrice != tc.wantTotal {
				t.Errorf("total = %v, want %v", trip.TotalPrice, tc.wantTotal)
			}
			if trip.Currency != "EUR" {
				t.Errorf("currency = %q, want EUR", trip.Currency)
			}
		})
	}
}

func TestCheapestWeekendTripMalformedDay(t *testing.T) {
	v := 10.0
	out := []ryanair.DailyFare{{Day: "not-a-date", Price: &v, Currency: "EUR"}}
	if _, err := ryanair.CheapestWeekendTrip(out, nil, 1, 3); err == nil {
		t.Fatal("expected error for malformed outbound day")
	}
}

// TestRoundTripNotTruncated locks in that roundTripFares is exempt from the
// oneWayFares truncation guard: the live endpoint returns nextPage=1 on every
// (even complete) response, so the guard must NOT fire there.
