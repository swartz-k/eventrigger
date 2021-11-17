package events

import (
	"context"
	"testing"
)

func Test_NewController(t *testing.T) {
	ctx := context.Background()
	controller, err := NewCloudEventController(9876)
	if err != nil {
		t.Fatal(err)
	}
	err = controller.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
