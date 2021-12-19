package server

import (
	"testing"
)

func Test_NewEventController(t *testing.T) {
	monitor, err := NewK8sEventsMonitor()
	if err != nil {
		t.Fatal(err)
	}
	err = monitor.Run()
	if err != nil {
		t.Fatal(err)
	}
}
