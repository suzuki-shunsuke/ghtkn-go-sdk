package deviceflow

import (
	"context"
	"time"
)

type Waiter interface {
	Wait(ctx context.Context, duration time.Duration) error
}

type SimpleWaiter struct{}

func (*SimpleWaiter) Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	}
}
