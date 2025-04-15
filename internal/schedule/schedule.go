package schedule

import "time"

type ScheduledJob struct {
	RunAt   time.Time
	Execute func()
}

func (s *ScheduledJob) Schedule() {
	go func() {
		delay := time.Until(s.RunAt)
		if delay > 0 {
			time.Sleep(delay)
		}
		s.Execute()
	}()
}
