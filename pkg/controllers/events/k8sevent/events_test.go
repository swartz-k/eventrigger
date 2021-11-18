package k8sevent

import (
	"context"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"testing"
)

func Test_NewEventController(t *testing.T) {
	ctx := context.Background()
	controller, err := NewEventController()
	if err != nil {
		t.Fatal(err)
	}
	monitorChannel := make(chan common.Monitor)
	err = controller.Run(ctx, monitorChannel)
	if err != nil {
		t.Fatal(err)
	}
}
