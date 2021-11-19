package k8sevent

import (
	"context"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"testing"
)

func Test_NewEventController(t *testing.T) {
	ctx := context.Background()
	stopChan := make(<-chan struct{})

	controller, err := NewEventController(stopChan)
	if err != nil {
		t.Fatal(err)
	}
	eventChannel := make(chan common.Event)
	err = controller.Run(ctx, eventChannel)
	if err != nil {
		t.Fatal(err)
	}
}
