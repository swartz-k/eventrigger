package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
)

type Interface interface {
	Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error
}
