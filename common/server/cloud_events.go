package server

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"net/http"
)

var (
	GlobalCloudEventsServer = NewCloudEventServer()
)

type cloudEventsServer struct {
	CTX      context.Context
	Receiver *client.EventReceiver
	// Channel Mapper
	EventChannelMapper map[string]*chan event.Event
}

func NewCloudEventServer() *cloudEventsServer {
	return &cloudEventsServer{CTX: context.Background(), EventChannelMapper: make(map[string]*chan event.Event)}
}

func (c *cloudEventsServer) Run(addr string) error {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		return errors.Wrap(err, "failed to create protocol")
	}

	c.Receiver, err = cloudevents.NewHTTPReceiveHandler(c.CTX, p, c.Receive)
	if err != nil {
		return err
	}

	zap.L().Info("cloud events server will listen on ", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, c.Receiver); err != nil {
		zap.L().Info("unable to start cloud events server ", zap.Error(err))
		return err
	}
	return nil
}

func (c *cloudEventsServer) Receive(cloudEvent cloudevents.Event) {
	// filter cloud events with register mapper
	key := UniqueCloudEventsKey(cloudEvent.Source(), cloudEvent.Type(), cloudEvent.SpecVersion())
	channel, ok := c.EventChannelMapper[key]
	if ok {
		comEvent := event.Event{
			Source:  cloudEvent.Source(),
			Type:    cloudEvent.Type(),
			Version: cloudEvent.SpecVersion(),
			Data:    string(cloudEvent.Data()),
		}
		zap.L().Info("cloud events receive ", zap.Any("event", cloudEvent))
		*channel <- comEvent
	} else {
		zap.L().Warn(fmt.Sprintf("cloud events receive but no monitor %s", cloudEvent))
	}
}

func UniqueCloudEventsKey(source, t, version string) string {
	return fmt.Sprintf("%s-%s-%s", source, t, version)
}

func (c *cloudEventsServer) UpdateMonitor(key string, channel *chan event.Event) {
	c.EventChannelMapper[key] = channel
}

func (c *cloudEventsServer) DeleteMonitor(key string) {
	delete(c.EventChannelMapper, key)
}
