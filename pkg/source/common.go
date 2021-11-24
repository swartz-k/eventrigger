package source

import (
	"context"
	"eventrigger.com/operator/common/event"
)

type Interface interface {
	Run(ctx context.Context, stopCh <- chan struct{}) <-chan event.Event
}
