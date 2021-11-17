package events

import (
	"context"
	"testing"
)

func Test_NewEventController(t *testing.T) {
	ctx := context.Background()
	controller, err := NewEventController()
	if err != nil {
		t.Fatal(err)
	}
	err = controller.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
