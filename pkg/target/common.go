package target

import (
	"context"
)

type Interface interface {
	Exec(ctx context.Context) error
}
