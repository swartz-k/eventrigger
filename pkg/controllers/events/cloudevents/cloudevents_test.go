package cloudevents

import (
	"context"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"testing"
)

func Test_NewController(t *testing.T) {
	ctx := context.Background()
	controller, err := NewCloudEventController(9876)
	if err != nil {
		t.Fatal(err)
	}
	monitorChannel := make(chan common.Monitor)
	err = controller.Run(ctx, monitorChannel)
	if err != nil {
		t.Fatal(err)
	}
}
