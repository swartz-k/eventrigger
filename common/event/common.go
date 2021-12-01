package event

import (
	"context"
	uuid2 "k8s.io/apimachinery/pkg/util/uuid"
	"strconv"
	"time"
)

type Event struct {
	Namespace string `json:"namespace" yaml:"namespace"`
	Source    string `json:"source" yaml:"source"`
	Type      string `json:"type" yaml:"type"`
	Version   string `json:"version" yaml:"version"`
	Data      string `json:"data" yaml:"data"`
	// Create Options
	UUID string `json:"uuid" yaml:"uuid"`
}

type Monitor struct {
	ActType   string `json:"act_type" yaml:"act_type"`   // ActType ADD DELETE
	Namespace string `json:"namespace" yaml:"namespace"` // opt, in k8s event needed
	Source    string `json:"source" yaml:"source"`
	Type      string `json:"type" yaml:"type"`
	Version   string `json:"version" yaml:"version"`
}

// Controller for listen and receive events like request, eg: cloud events、kubernetes events
type Controller interface {
	Run(ctx context.Context, eventChannel chan Event) error
}

func NewSimpleEvent(t, source, data string) Event {
	event := Event{
		Source: source,
		Type:   t,
		Data:   data,
		UUID:   string(uuid2.NewUUID()),
	}
	return event
}

func NewEvent(namespace, eventType, source, version, data, uuid string) Event {
	if uuid == "" {
		uuid = string(uuid2.NewUUID())
	}
	if version == "" {
		version = strconv.Itoa(int(time.Now().UnixNano()))
	}
	event := Event{
		Namespace: namespace,
		Source:    source,
		Type:      eventType,
		Version:   version,
		Data:      data,
		UUID:      uuid,
	}
	return event
}
