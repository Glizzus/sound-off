package schedule_test

import (
	"testing"
	"time"

	"github.com/glizzus/sound-off/internal/schedule"
)

func TestNextRunTimesAfterSuccess(t *testing.T) {
	table := []struct {
		cron  string
		after time.Time
		n     int
		want  []time.Time
	}{
		{
			cron:  "0 0 * * *", // Every day at midnight
			after: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			n:     3,
			want: []time.Time{
				time.Date(2023, 10, 2, 0, 0, 0, 0, time.UTC),
				time.Date(2023, 10, 3, 0, 0, 0, 0, time.UTC),
				time.Date(2023, 10, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			cron:  "*/5 * * * *", // Every 5 minutes
			after: time.Date(1981, 8, 29, 12, 0, 0, 0, time.UTC),
			n:     4,
			want: []time.Time{
				time.Date(1981, 8, 29, 12, 5, 0, 0, time.UTC),
				time.Date(1981, 8, 29, 12, 10, 0, 0, time.UTC),
				time.Date(1981, 8, 29, 12, 15, 0, 0, time.UTC),
				time.Date(1981, 8, 29, 12, 20, 0, 0, time.UTC),
			},
		},
		{
			cron:  "@monthly", // Monthly on the first weekday of the month at 00:00
			after: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			n:     2,
			want: []time.Time{
				time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			cron:  "0 9 * * 1", // Every Monday at 9 AM
			after: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			n:     3,
			want: []time.Time{
				time.Date(2023, 10, 2, 9, 0, 0, 0, time.UTC),
				time.Date(2023, 10, 9, 9, 0, 0, 0, time.UTC),
				time.Date(2023, 10, 16, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range table {
		t.Run(tc.cron, func(t *testing.T) {
			got, err := schedule.NextRunTimesAfter(tc.cron, tc.after, tc.n)
			if err != nil {
				t.Fatalf("NextRunTimesAfter(%q, %v, %d) returned error: %v", tc.cron, tc.after, tc.n, err)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("NextRunTimesAfter(%q, %v, %d) = %v; want %v", tc.cron, tc.after, tc.n, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNextRunTimesAfterFailure(t *testing.T) {
	table := []struct {
		cron  string
		after time.Time
		n     int
	}{
		{
			cron:  "invalid cron",
			after: time.Now(),
			n:     3,
		},
		{
			cron:  "0 0 * * *",
			after: time.Now(),
			n:     -1,
		},
	}

	for _, tc := range table {
		t.Run(tc.cron, func(t *testing.T) {
			got, err := schedule.NextRunTimesAfter(tc.cron, tc.after, tc.n)
			if err == nil {
				t.Fatalf("NextRunTimesAfter(%q, %v, %d) expected error but got result: %v", tc.cron, tc.after, tc.n, got)
			}
		})
	}
}
