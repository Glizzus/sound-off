package schedule

import (
	"context"
	"time"
)

func RunAt(ctx context.Context, runAt time.Time, execute func(ctx context.Context)) {
	go func() {
		delay := time.Until(runAt)
		if delay > 0 {
			time.Sleep(delay)
		}
		execute(ctx)
	}()
}