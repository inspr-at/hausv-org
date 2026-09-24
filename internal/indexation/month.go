package indexation

import (
	"fmt"
	"time"
)

// Month is an ISO calendar month (YYYY-MM). Empty means no month, not January.
type Month string

func ParseMonth(s string) (Month, error) {
	_, err := time.Parse("2006-01", s)
	if err != nil || len(s) != 7 || s[:4] == "0000" {
		return "", fmt.Errorf("invalid month %q", s)
	}
	return Month(s), nil
}

func (m Month) Valid() bool {
	_, err := ParseMonth(string(m))
	return err == nil
}

func (m Month) date() time.Time {
	d, _ := time.Parse("2006-01", string(m))
	return d
}

func monthOf(t time.Time) Month { return Month(t.Format("2006-01")) }
func (m Month) next() Month     { return monthOf(m.date().AddDate(0, 1, 0)) }

// civilDate preserves the caller's calendar date; it does not reinterpret a
// Vienna midnight as the previous day in UTC. Calculations are timezone-free.
func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
