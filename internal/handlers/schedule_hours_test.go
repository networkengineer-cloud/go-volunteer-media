package handlers

import "testing"

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
