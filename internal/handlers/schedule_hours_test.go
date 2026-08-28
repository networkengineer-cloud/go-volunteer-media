package handlers

import (
	"testing"
	"time"
)

func TestMaxHourFor(t *testing.T) {
	tests := []struct {
		name      string
		dayOfWeek int
		want      int
	}{
		{"Sunday caps at 15", 0, 15},
		{"Monday caps at 17", 1, 17},
		{"Friday caps at 17", 5, 17},
		{"Saturday caps at 15", 6, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maxHourFor(tt.dayOfWeek); got != tt.want {
				t.Errorf("maxHourFor(%d) = %d, want %d", tt.dayOfWeek, got, tt.want)
			}
		})
	}
}

func TestSlotDurationMinutes(t *testing.T) {
	tests := []struct {
		name      string
		dayOfWeek int
		hour      int
		want      int
	}{
		{"weekday normal hour is 60 min", 3, 14, 60},
		{"weekday terminal hour (17) is 90 min", 3, 17, 90},
		{"weekend normal hour is 60 min", 0, 10, 60},
		{"weekend terminal hour (15) is 90 min", 6, 15, 90},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slotDurationMinutes(tt.dayOfWeek, tt.hour); got != tt.want {
				t.Errorf("slotDurationMinutes(%d, %d) = %d, want %d", tt.dayOfWeek, tt.hour, got, tt.want)
			}
		})
	}
}

func TestWeekParity(t *testing.T) {
	tests := []struct {
		name string
		date string // any date; will be truncated to that week's Sunday internally by the test via weekStartOf
		want string
	}{
		{"the reference Sunday itself is A", "2024-01-07", "a"},
		{"one week later is B", "2024-01-14", "b"},
		{"two weeks later is A again", "2024-01-21", "a"},
		{"one week before the reference is B", "2023-12-31", "b"},
		{"52 weeks later (even) is A", "2025-01-05", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := time.Parse("2006-01-02", tt.date)
			if err != nil {
				t.Fatalf("bad test date: %v", err)
			}
			if got := weekParity(weekStartOf(d)); got != tt.want {
				t.Errorf("weekParity(weekStartOf(%s)) = %q, want %q", tt.date, got, tt.want)
			}
		})
	}
}

func TestSlotActiveForWeek(t *testing.T) {
	aWeek, _ := time.Parse("2006-01-02", "2024-01-07")  // parity "a"
	bWeek, _ := time.Parse("2006-01-02", "2024-01-14") // parity "b"

	tests := []struct {
		name     string
		cadence  string
		weekStart time.Time
		want     bool
	}{
		{"weekly is active on an A week", "weekly", aWeek, true},
		{"weekly is active on a B week", "weekly", bWeek, true},
		{"biweekly_a is active on an A week", "biweekly_a", aWeek, true},
		{"biweekly_a is inactive on a B week", "biweekly_a", bWeek, false},
		{"biweekly_b is inactive on an A week", "biweekly_b", aWeek, false},
		{"biweekly_b is active on a B week", "biweekly_b", bWeek, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slotActiveForWeek(tt.cadence, tt.weekStart); got != tt.want {
				t.Errorf("slotActiveForWeek(%q, %v) = %v, want %v", tt.cadence, tt.weekStart, got, tt.want)
			}
		})
	}
}
