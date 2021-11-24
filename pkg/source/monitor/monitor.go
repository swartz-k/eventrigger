package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	"go.uber.org/zap"
)

type Runner interface {
	Run(eventChannel chan event.Event, stopCh <- chan struct{}) error
}

type Manager interface {
	Run() error
}

type manager struct {
	CTX            context.Context
	// enable ingress proxy
	EnableProxy bool
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
	if m.EnableProxy {
		go m.ListenProxy()
	}
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
			mqttCfg := monitor.Template.MQTT
			opts := &MQTTOptions{
				URI: mqttCfg.URL,
				Topic: mqttCfg.Topic,
				Username: mqttCfg.Username,
				Password: mqttCfg.Password,
			}
			runner, err = NewMQTTRunner(opts)
			if err != nil {
				zap.L().Info("monitor name or template is nil")
			}
			m.MonitorMapper[fmt.Sprintf("%s", monitor.Name)] = runner
		}
	}
}
