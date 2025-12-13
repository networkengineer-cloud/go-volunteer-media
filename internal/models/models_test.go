package models

import (
	"testing"
	"time"
)

func TestAnimal_LengthOfStay(t *testing.T) {
	tests := []struct {
		name        string
		arrivalDate *time.Time
		expected    int
	}{
		{
			name:        "no arrival date",
			arrivalDate: nil,
			expected:    0,
		},
		{
			name: "arrived today",
			arrivalDate: func() *time.Time {
				now := time.Now()
				return &now
			}(),
			expected: 0,
		},
		{
			name: "arrived 1 day ago",
			arrivalDate: func() *time.Time {
				yesterday := time.Now().AddDate(0, 0, -1)
				return &yesterday
			}(),
			expected: 1,
		},
		{
			name: "arrived 7 days ago",
			arrivalDate: func() *time.Time {
				weekAgo := time.Now().AddDate(0, 0, -7)
				return &weekAgo
			}(),
			expected: 7,
		},
		{
			name: "arrived 30 days ago",
			arrivalDate: func() *time.Time {
				monthAgo := time.Now().AddDate(0, 0, -30)
				return &monthAgo
			}(),
			expected: 30,
		},
		{
			name: "arrived 365 days ago",
			arrivalDate: func() *time.Time {
				yearAgo := time.Now().AddDate(0, 0, -365)
				return &yearAgo
			}(),
			expected: 365,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animal := &Animal{
				ArrivalDate: tt.arrivalDate,
			}
			result := animal.LengthOfStay()
			if result != tt.expected {
				t.Errorf("LengthOfStay() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestAnimal_CurrentStatusDuration(t *testing.T) {
	tests := []struct {
		name             string
		lastStatusChange *time.Time
		expected         int
	}{
		{
			name:             "no status change",
			lastStatusChange: nil,
			expected:         0,
		},
		{
			name: "status changed today",
			lastStatusChange: func() *time.Time {
				now := time.Now()
				return &now
			}(),
			expected: 0,
		},
		{
			name: "status changed 1 day ago",
			lastStatusChange: func() *time.Time {
				yesterday := time.Now().AddDate(0, 0, -1)
				return &yesterday
			}(),
			expected: 1,
		},
		{
			name: "status changed 10 days ago",
			lastStatusChange: func() *time.Time {
				tenDaysAgo := time.Now().AddDate(0, 0, -10)
				return &tenDaysAgo
			}(),
			expected: 10,
		},
		{
			name: "status changed 45 days ago",
			lastStatusChange: func() *time.Time {
				ago := time.Now().AddDate(0, 0, -45)
				return &ago
			}(),
			expected: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animal := &Animal{
				LastStatusChange: tt.lastStatusChange,
			}
			result := animal.CurrentStatusDuration()
			if result != tt.expected {
				t.Errorf("CurrentStatusDuration() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestAnimal_QuarantineEndDate(t *testing.T) {
	tests := []struct {
		name                string
		quarantineStartDate *time.Time
		checkResult         func(*testing.T, *time.Time)
	}{
		{
			name:                "no quarantine start date",
			quarantineStartDate: nil,
			checkResult: func(t *testing.T, result *time.Time) {
				if result != nil {
					t.Error("Expected nil for no quarantine start date")
				}
			},
		},
		{
			name: "quarantine ends on a weekday",
			quarantineStartDate: func() *time.Time {
				// Set to a Monday so 10 days later is a Thursday
				date := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC) // Monday
				return &date
			}(),
			checkResult: func(t *testing.T, result *time.Time) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				// Should be Thursday, 10 days later
				expected := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("Expected %v, got %v", expected, result)
				}
				if result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
					t.Error("End date should not be on a weekend")
				}
			},
		},
		{
			name: "quarantine would end on Saturday - adjust to Monday",
			quarantineStartDate: func() *time.Time {
				// Set to a Wednesday so 10 days later is a Saturday
				date := time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC) // Wednesday
				return &date
			}(),
			checkResult: func(t *testing.T, result *time.Time) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				// 10 days from Wednesday would be Saturday, should adjust to Monday
				// Nov 5 (Wed) + 10 days = Nov 15 (Sat) → Nov 17 (Mon)
				if result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
					t.Error("End date should not be on a weekend")
				}
				if result.Weekday() != time.Monday {
					t.Errorf("Expected Monday when 10 days ends on Saturday, got %v", result.Weekday())
				}
			},
		},
		{
			name: "quarantine would end on Sunday - adjust to Monday",
			quarantineStartDate: func() *time.Time {
				// Set to a Thursday so 10 days later is a Sunday
				date := time.Date(2025, 11, 6, 0, 0, 0, 0, time.UTC) // Thursday
				return &date
			}(),
			checkResult: func(t *testing.T, result *time.Time) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				// 10 days from Thursday would be Sunday, should adjust to Monday
				if result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
					t.Error("End date should not be on a weekend")
				}
				if result.Weekday() != time.Monday {
					t.Errorf("Expected Monday when 10 days ends on Sunday, got %v", result.Weekday())
				}
			},
		},
		{
			name: "quarantine start date in the past",
			quarantineStartDate: func() *time.Time {
				// 20 days ago
				date := time.Now().AddDate(0, 0, -20)
				return &date
			}(),
			checkResult: func(t *testing.T, result *time.Time) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				// Should be 10 days after start date
				if result.Before(time.Now()) {
					// That's fine, quarantine should already be over
				}
				if result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
					t.Error("End date should not be on a weekend")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animal := &Animal{
				QuarantineStartDate: tt.quarantineStartDate,
			}
			result := animal.QuarantineEndDate()
			tt.checkResult(t, result)
		})
	}
}

func TestAnimal_MethodsWithRealData(t *testing.T) {
	// Test with a realistic animal scenario
	t.Run("realistic shelter animal", func(t *testing.T) {
		arrivalDate := time.Now().AddDate(0, 0, -15)    // Arrived 15 days ago
		statusChange := time.Now().AddDate(0, 0, -5)    // Status changed 5 days ago
		quarantineStart := time.Now().AddDate(0, 0, -7) // Quarantine started 7 days ago

		animal := &Animal{
			Name:                "Buddy",
			Species:             "Dog",
			Status:              "bite_quarantine",
			ArrivalDate:         &arrivalDate,
			LastStatusChange:    &statusChange,
			QuarantineStartDate: &quarantineStart,
		}

		lengthOfStay := animal.LengthOfStay()
		if lengthOfStay != 15 {
			t.Errorf("Expected length of stay to be 15 days, got %d", lengthOfStay)
		}

		statusDuration := animal.CurrentStatusDuration()
		if statusDuration != 5 {
			t.Errorf("Expected status duration to be 5 days, got %d", statusDuration)
		}

		endDate := animal.QuarantineEndDate()
		if endDate == nil {
			t.Fatal("Expected quarantine end date to be set")
		}
		if endDate.Weekday() == time.Saturday || endDate.Weekday() == time.Sunday {
			t.Error("Quarantine end date should not be on a weekend")
		}
	})
}
