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

// digestMarketQuoteDay returns the US Eastern session date for the last regular
// close before the morning digest cutoff (07:00 GMT+7 on calendarDay).
func digestMarketQuoteDay(calendarDay time.Time) time.Time {
	_, cutoff := morningDigestWindow(calendarDay.In(vnTimezone))
	loc := usEastern
	anchor := cutoff.In(loc)
	for offset := 0; offset < 10; offset++ {
		day := anchor.AddDate(0, 0, -offset)
		switch day.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		}
		sessionClose := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, loc)
		if sessionClose.Before(cutoff) {
			return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
		}
	}
	return normalizeTradingDay(calendarDay).In(loc)
}

func formatQuoteSessionLabel(quoteDay time.Time) string {
	return quoteDay.In(usEastern).Format("02/01/2006") + " (Mỹ)"
}