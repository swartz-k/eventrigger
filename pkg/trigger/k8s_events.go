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

type K8sEventsOptions struct {
	Namespace  string
	APIVersion string
	Type       string
	Kind       string
}

type K8sEventsTrigger struct {
	Opts         *K8sEventsOptions
	Key          string
	EventChannel *chan event.Event
}

func parseK8sEventsMeta(meta map[string]string) (opts *K8sEventsOptions, err error) {
	opts = &K8sEventsOptions{}

	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, errors.Wrap(err, "parse k8s events trigger")
	}

	return opts, nil
}

func NewK8sEventsTrigger(meta map[string]string) (*K8sEventsTrigger, error) {
	opts, err := parseK8sEventsMeta(meta)
	if err != nil {
		return nil, err
	}
	m := &K8sEventsTrigger{
		Opts: opts,
		Key:  server.UniqueK8sEventKey(opts.Kind, opts.Type, opts.APIVersion, opts.Namespace),
	}

	return m, nil
}

func (m *K8sEventsTrigger) Run(ctx context.Context, eventChannel chan event.Event) error {
	m.EventChannel = &eventChannel
	server.GlobalK8sEventsMonitor.UpdateMonitor(m.Key, &eventChannel)
	zap.L().Debug(fmt.Sprintf("k8s events trigger add monitor with event: %s exist", m.Key))
	return nil
}

func (m *K8sEventsTrigger) Stop() error {
	server.GlobalK8sEventsMonitor.DeleteMonitor(m.Key)

	return nil
}
