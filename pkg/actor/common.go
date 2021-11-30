package actor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"time"
)

type Interface interface {
	String() string
	Exec(ctx context.Context, event event.Event) error

	// GetScaleToZeroTime scale to zero
	GetScaleToZeroTime() *time.Duration
	ScaleToZero(ctx context.Context) error
}
