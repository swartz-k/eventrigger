package manager

import (
	"context"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/pkg/actor"
	"eventrigger.com/operator/pkg/actor/k8s"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/monitor"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type RunnerInterface interface {

	Run(stopCh chan struct{}) error
}

type runner struct {
	CTX context.Context
	eventCh chan event.Event
	stopCh chan struct{}
	Monitor monitor.Interface
	Actor  actor.Interface
}

func ParseSensorMonitor(m *v1.Monitor) (source monitor.Interface, err error) {
	if m == nil || m.Template == nil {
		return nil, errors.New(fmt.Sprintf("%+v monitor or template is nil parse failed", m))
	}
	if m.Template.MQTT != nil {
		source := m.Template.MQTT
		opts := &monitor.MQTTOptions{
			URI: source.URL,
			Topic: source.Topic,
			Username: source.Username,
			Password: source.Password,
		}
		return monitor.NewMQTTMonitor(opts)
	}
	return nil, nil
}

func ParseSensorTrigger(trigger *v1.Trigger) (actor actor.Interface, err error) {
	if trigger.Template == nil {
		return nil, errors.New("trigger template is nil")
	}
	if trigger.Template.K8s != nil {
		if trigger.Template.K8s.Source == nil {
			return nil, errors.New("init k8s actor failed, source is nil")
		}
		return k8s.NewK8SActor(string(trigger.Template.K8s.Operation), trigger.Template.K8s.Source.Resource)
	}

	if trigger.Template.HTTP != nil {
		return nil, errors.New("temp not support")
	}
	return nil, errors.New("no valid template")
}

func NewRunner(sensor *v1.Sensor) (r RunnerInterface, err error){
	ctx := context.Background()
	if sensor == nil {
		return nil, errors.New("sensor is nil, runner failed")
	}
	monitor, err := ParseSensorMonitor(&sensor.Spec.Monitor)
	if err != nil {
		return nil, err
	}
	actor, err := ParseSensorTrigger(&sensor.Spec.Trigger)
	if err != nil {
		return nil, err
	}
	r = &runner{
		CTX: ctx,
		Monitor: monitor,
		Actor: actor,
		eventCh: make(chan event.Event),
	}
	return r, nil
}

func (r *runner) Run(stopCh chan struct{}) error {
	err := r.Monitor.Run(r.CTX, r.eventCh, stopCh)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-r.eventCh:
			zap.L().Info(fmt.Sprintf("receive event %s-%s, exec actor", event.Type, event.Source))
			err := r.Actor.Exec(r.CTX, event)
			if err != nil {
				err = errors.Wrapf(err, "actor exec with event %s-%s", event.Type, event.Source)
				zap.L().Error("", zap.Error(err))
			} else {
				zap.L().Info(fmt.Sprintf("successfully exec event %s-%s with actor", event.Type, event.Source))
			}
		case <-stopCh:
			zap.L().Warn("receive stop channel, stop!!!")
			return nil
		}
	}
	return nil
}