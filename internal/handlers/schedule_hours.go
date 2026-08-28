package handlers

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
