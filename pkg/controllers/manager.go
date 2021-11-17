package controllers

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/sync/errsgroup"
	"eventrigger.com/operator/pkg/controllers/events"
	"fmt"
	"k8s.io/client-go/rest"
)

type ManagerOptions struct {
	CloudEventsPort     uint `json:"cloud_events_port" yaml:"cloud_events_port"`
	OperatorPort        int  `json:"operator_port" yaml:"operator_port"`
	OperatorMetricsPort int  `json:"metrics_addr"`
	OperatorHealthPort  int  `json:"health_port"`
	LeaderElect         bool `json:"leader_elect"`
}

type Manager struct {
	Cfg     *rest.Config
	Options *ManagerOptions
	stopCh  <-chan struct{}
}

func NewManager(options *ManagerOptions) *Manager {
	if options == nil {
		options = &ManagerOptions{
			CloudEventsPort: consts.DefaultCloudEventsControllerPort,
		}
	}
	m := &Manager{Options: options}
	return m
}

func (m *Manager) Run() error {
	ctx := context.Background()
	cloudEventsController, err := events.NewCloudEventController(m.Options.CloudEventsPort)
	if err != nil {
		return err
	}

	eventsController, err := events.NewEventController()
	if err != nil {
		return err
	}

	opts := OperatorOptions{
		Port:        m.Options.OperatorPort,
		MetricsAddr: fmt.Sprintf(":%d", m.Options.OperatorMetricsPort),
		HealthAddr:  fmt.Sprintf(":%d", m.Options.OperatorMetricsPort),
		LeaderElect: m.Options.LeaderElect,
	}
	operator, err := NewOperator(opts)
	if err != nil {
		return err
	}

	var eg errsgroup.Group
	eg.Go(func() error {
		return cloudEventsController.Run(ctx, m.Events2SensorChannel)
	})

	eg.Go(func() error {
		return eventsController.Run(ctx, m.Events2SensorChannel)
	})

	eg.Go(func() error {
		return operator.Run(ctx, m.Sensor2Events)
	})

	return eg.WaitWithStopChannel(m.stopCh)
}
