package handlers

import "time"

// isWeekendDay reports whether dayOfWeek (0=Sunday..6=Saturday) falls on a
// weekend, which closes earlier than a weekday.
func isWeekendDay(dayOfWeek int) bool {
	return dayOfWeek == 0 || dayOfWeek == 6
}

// maxHourFor returns the last valid ShiftSlot start hour for the given
// day-of-week: 15 (3pm, a 90-min slot ending 4:30pm) on weekends, 17 (5pm, a
// 90-min slot ending 6:30pm) on weekdays.
func maxHourFor(dayOfWeek int) int {
	if isWeekendDay(dayOfWeek) {
		return 15
	}
	return 17
}

// slotDurationMinutes returns how long a slot starting at hour on the given
// day-of-week runs: 90 minutes for the day's terminal hour, 60 otherwise.
func slotDurationMinutes(dayOfWeek, hour int) int {
	if hour == maxHourFor(dayOfWeek) {
		return 90
	}
	return 60
}

// biweeklyReferenceSunday anchors the "week A" / "week B" cycle for
// biweekly cadences. Must stay a Sunday and must never change once any
// biweekly slot exists in production, or every existing A/B assignment
// flips. Mirrored exactly in frontend/src/pages/group/scheduleGrid.ts.
var biweeklyReferenceSunday = time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)

// weekStartOf returns the Sunday (UTC midnight) that starts date's week.
func weekStartOf(date time.Time) time.Time {
	d := date.UTC().Truncate(24 * time.Hour)
	return d.AddDate(0, 0, -int(d.Weekday()))
}

// weekParity classifies weekStart (must already be a Sunday, e.g. from
// weekStartOf or parseWeekStart) as week "a" or "b", counting whole weeks
// since biweeklyReferenceSunday. The double-mod handles weeks before the
// reference date, where Go's % can return a negative remainder.
func weekParity(weekStart time.Time) string {
	weeks := int(weekStart.Sub(biweeklyReferenceSunday).Hours() / (24 * 7))
	if ((weeks%2)+2)%2 == 0 {
		return "a"
	}
	return "b"
}

// slotActiveForWeek reports whether a ShiftSlot with the given cadence
// occurs during the week starting at weekStart. "weekly" is always active;
// "biweekly_a"/"biweekly_b" are active only on their matching parity week.
func slotActiveForWeek(cadence string, weekStart time.Time) bool {
	switch cadence {
	case "biweekly_a":
		return weekParity(weekStart) == "a"
	case "biweekly_b":
		return weekParity(weekStart) == "b"
	default:
		return true
	}
}
