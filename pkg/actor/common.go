package actor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"time"
)

type Interface interface {
	String() string
	Exec(ctx context.Context, event event.Event) error

	// GetTickerTime to do check
	GetTickerTime() time.Duration
	Check(ctx context.Context, nowTime time.Time, lastEvent time.Time) error
}
