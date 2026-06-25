package worldnews

import "time"

// MorningDigestHour — mốc 7:00 sáng (GMT+7) cho bản tin tổng hợp.
const MorningDigestHour = 7

func morningDigestWindow(calendarDay time.Time) (time.Time, time.Time) {
	day := calendarDay.In(vnTimezone)
	until := time.Date(day.Year(), day.Month(), day.Day(), MorningDigestHour, 0, 0, 0, vnTimezone)
	since := until.Add(-24 * time.Hour)
	return since, until
}

func formatDigestWindow(since, until time.Time) string {
	return since.In(vnTimezone).Format("02/01/2006 15:04") +
		" – " +
		until.In(vnTimezone).Format("02/01/2006 15:04")
}

func formatDigestUntil(until time.Time) string {
	return until.In(vnTimezone).Format("02/01/2006 15:04") + " (GMT+7)"
}