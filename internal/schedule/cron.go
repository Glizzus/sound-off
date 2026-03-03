package schedule

import (
	"fmt"
	"time"

	"github.com/hashicorp/cronexpr"
)

// NextRunTimes returns the next N run times that a cron expression will run,
// evaluated in the given location.
func NextRunTimes(cron string, loc *time.Location, n int) ([]time.Time, error) {
	cutoff := time.Now().In(loc)
	return NextRunTimesAfter(cron, cutoff, n)
}

// NextRunTimesAfter returns the next N run times after a specific time.
// It returns an error if the cron expression is invalid or if count is less than 1.
func NextRunTimesAfter(cron string, after time.Time, n int) ([]time.Time, error) {
	if n <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}
	expr, err := cronexpr.Parse(cron)
	if err != nil {
		return nil, err
	}
	return expr.NextN(after, uint(n)), nil
}

func ValidateCron(cron string) error {
	_, err := cronexpr.Parse(cron)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}
