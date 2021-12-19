package trigger

import (
	"context"
	"eventrigger.com/operator/common/event"
)

type Interface interface {
	Run(ctx context.Context, eventChannel chan event.Event) error
	Stop() error
}
