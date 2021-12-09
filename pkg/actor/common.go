package actor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"time"
)

type Interface interface {
	String() string
	Exec(ctx context.Context, event event.Event) error

	Check(ctx context.Context, scaleTime time.Duration, lastEvent time.Time) error
}
