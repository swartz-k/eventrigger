package events

import "context"

type Event struct {

}


// Controller for listen and receive events like request, eg: cloud events、kubernetes events
type Controller interface {
	Run(ctx context.Context) error
}
