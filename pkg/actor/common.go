package actor

import (
	"context"
	"eventrigger.com/operator/common/event"
)

type Interface interface {
	Run(stopCh <- chan struct{}) error
	Exec(ctx context.Context, event event.Event) error
}
