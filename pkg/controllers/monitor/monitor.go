package monitor

import (
	"context"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	"go.uber.org/zap"
)

type Runner interface {
}

type Manager interface {
	Run() error
}

type manager struct {
	CTX            context.Context
	StopCh         <-chan struct{}
	MonitorChannel chan v1.Monitor
	MonitorMapper  map[string]Runner
}

func NewManager(ctx context.Context, stopChan <-chan struct{}, monitorChannel chan v1.Monitor) (Manager, error) {
	r := &manager{
		CTX:            ctx,
		MonitorChannel: monitorChannel,
		StopCh:         stopChan,
	}

	return r, nil
}

func (m *manager) Run() error {
	go m.ReceiverMonitor()
	return nil
}

func (m *manager) ReceiverMonitor() {
	for monitor := range m.MonitorChannel {
		zap.L().Info("receive monitor", zap.String("name", monitor.Name))
		var runner Runner
		var err error
		if monitor.Name == "" || monitor.Template == nil {
			zap.L().Info("monitor name or template is nil")
			break
		}
		if monitor.Template.MQTT != nil {
			runner, err = NewMQTTRunner(monitor.Template.MQTT)
			if err != nil {
				zap.L().Info("monitor name or template is nil")
			}
			m.MonitorMapper[fmt.Sprintf("%s", monitor.Name)] = runner
		}
	}
}
