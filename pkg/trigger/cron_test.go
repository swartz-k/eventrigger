package trigger

import (
	"context"
	"eventrigger.com/operator/common/event"
	"testing"
	"time"
)

func TestCronRunner(t *testing.T) {
	meta := map[string]string{
		"cron": "*/1 * * * * *",
	}
	runner, err := NewCronMonitor(meta)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	stopCh := make(chan struct{})
	eventCh := make(chan event.Event)

	go runner.Run(ctx, eventCh)

	go func() {
		for {
			select {
			case event := <-eventCh:
				t.Logf("receive event %+v", event)
			}
		}
	}()

	time.Sleep(3 * time.Second)
	stopCh <- struct{}{}
	time.Sleep(7 * time.Second)
	t.Log("run end")
}

func TestChannel(t *testing.T) {
	stopCh := make(chan struct{})
	go func() {
		select {
		case <-stopCh:
			t.Log("receive stop ch")
		}
	}()

	time.Sleep(1 * time.Second)
	stopCh <- struct{}{}
}
