package trigger

import (
	"context"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/server"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type CloudEventsOptions struct {
	Namespace  string
	APIVersion string
	Kind       string
	Type       string
}

type CloudEventsTrigger struct {
	Opts         *CloudEventsOptions
	Key          string
	EventChannel *chan event.Event
}

func parseCloudEventsMeta(meta map[string]string) (opts *CloudEventsOptions, err error) {
	opts = &CloudEventsOptions{}

	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, errors.Wrap(err, "parse k8s events trigger")
	}

	return opts, nil
}

func NewCloudEventsTrigger(meta map[string]string) (*CloudEventsTrigger, error) {
	opts, err := parseCloudEventsMeta(meta)
	if err != nil {
		return nil, err
	}
	m := &CloudEventsTrigger{
		Opts: opts,
		Key:  server.UniqueK8sEventKey(opts.Kind, opts.Type, opts.APIVersion, opts.Namespace),
	}

	return m, nil
}

func (m *CloudEventsTrigger) Run(ctx context.Context, eventChannel chan event.Event) error {
	m.EventChannel = &eventChannel
	server.GlobalK8sEventsMonitor.UpdateMonitor(m.Key, &eventChannel)
	zap.L().Debug(fmt.Sprintf("k8s events trigger add monitor with event: %s exist", m.Key))
	return nil
}

func (m *CloudEventsTrigger) Stop() error {
	server.GlobalK8sEventsMonitor.DeleteMonitor(m.Key)

	return nil
}
