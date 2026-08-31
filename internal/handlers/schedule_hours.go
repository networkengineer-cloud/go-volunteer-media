package handlers

import (
	"fmt"
	"strings"
	"time"
)

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

// formatClockAMPM renders an hour (0-23) and minute as a 12-hour clock
// string, e.g. formatClockAMPM(18, 30) -> "6:30 PM". Mirrors
// frontend/src/pages/group/scheduleGrid.ts's formatClock.
func formatClockAMPM(hour, minute int) string {
	period := "AM"
	if hour >= 12 {
		period = "PM"
	}
	displayHour := hour % 12
	if displayHour == 0 {
		displayHour = 12
	}
	return fmt.Sprintf("%d:%02d %s", displayHour, minute, period)
}

// formatSlotRangeLabel renders a ShiftSlot/CoverageRequest's day-of-week and
// start hour as a plain start-time label ("10:00 AM") for a normal 60-min
// slot, or a "start–end" range ("5:00–6:30 PM") for the day's terminal
// 90-min slot - matching frontend/src/pages/group/scheduleGrid.ts's
// formatSlotRangeLabel exactly (keep both in sync if the hour rules ever
// change).
func formatSlotRangeLabel(dayOfWeek, hour int) string {
	duration := slotDurationMinutes(dayOfWeek, hour)
	if duration == 60 {
		return formatClockAMPM(hour, 0)
	}
	endTotalMinutes := hour*60 + duration
	endHour := endTotalMinutes / 60
	endMinute := endTotalMinutes % 60
	startTime := strings.TrimSuffix(strings.TrimSuffix(formatClockAMPM(hour, 0), " AM"), " PM")
	endTime := formatClockAMPM(endHour, endMinute)
	return fmt.Sprintf("%s–%s", startTime, endTime)
}

// formatHourRangeLabel renders a contiguous run of hours [firstHour,
// lastHour] on the given day-of-week as a "start–end" range, e.g.
// formatHourRangeLabel(2, 8, 10) -> "8:00–11:00 AM". The end time accounts
// for the terminal slot's 90-minute duration if lastHour happens to be
// that day's terminal hour. Unlike formatSlotRangeLabel, always renders a
// full range (even for a single hour) since this is used to summarize a
// reassignment notification, where the actual end time matters regardless
// of whether it's the day's 60- or 90-minute slot.
func formatHourRangeLabel(dayOfWeek, firstHour, lastHour int) string {
	duration := slotDurationMinutes(dayOfWeek, lastHour)
	endTotalMinutes := lastHour*60 + duration
	endHour := endTotalMinutes / 60
	endMinute := endTotalMinutes % 60
	startTime := strings.TrimSuffix(strings.TrimSuffix(formatClockAMPM(firstHour, 0), " AM"), " PM")
	endTime := formatClockAMPM(endHour, endMinute)
	return fmt.Sprintf("%s–%s", startTime, endTime)
}

// formatReassignedHoursSummary groups sortedHours (ascending, deduplicated)
// into contiguous runs and renders each via formatHourRangeLabel, joining
// multiple runs with " and " - e.g. hours [8,9,10,13] on a weekday ->
// "8:00–11:00 AM and 1:00–2:00 PM". Used so a multi-hour reassignment
// notification describes the whole span in one sentence instead of
// enumerating every hour.
func formatReassignedHoursSummary(dayOfWeek int, sortedHours []int) string {
	if len(sortedHours) == 0 {
		return ""
	}
	var runs []string
	runStart := sortedHours[0]
	prev := sortedHours[0]
	for _, h := range sortedHours[1:] {
		if h == prev+1 {
			prev = h
			continue
		}
		runs = append(runs, formatHourRangeLabel(dayOfWeek, runStart, prev))
		runStart = h
		prev = h
	}
	runs = append(runs, formatHourRangeLabel(dayOfWeek, runStart, prev))
	return strings.Join(runs, " and ")
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
