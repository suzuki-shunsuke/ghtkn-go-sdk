package deviceflow

import (
	"context"
	"time"
)

type waiter interface {
	Wait(ctx context.Context, duration time.Duration) error
}

type simpleWaiter struct{}

func (*simpleWaiter) Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	}
}
