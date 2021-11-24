package manager

import (
	"context"
	"eventrigger.com/operator/pkg/actor"
	"eventrigger.com/operator/pkg/actor/k8s"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/source"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type RunnerInterface interface {
	Run(stopCh chan struct{})
}

type runner struct {
	CTX context.Context
	stopCh chan struct{}
	Source source.Interface
	Actor actor.Interface
}

func ParseSensorSource(monitor v1.Monitor) (source source.Interface, err error) {

	return nil, nil
}

func ParseSensorTrigger(trigger v1.Trigger) (actor actor.Interface, err error) {
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
	if sensor == nil {
		return nil, errors.New("sensor is nil, runner failed")
	}
	source, err := ParseSensorSource(sensor.Spec.Monitor)
	actor, err := ParseSensorTrigger(sensor.Spec.Trigger)
	r = &runner{
		Source: source,
		Actor: actor,
	}
	return r, nil
}

func (r *runner) Run(stopCh chan struct{}) {
	select {
	case event := <- r.Source.Run(r.CTX, stopCh):
		err := r.Actor.Exec(r.CTX, event)
		if err != nil {
			zap.L().Warn("receive stop channel, stop!!!")
			break
		}
		zap.L().Info(fmt.Sprintf("successfully exec event %s-%s with actor", event.Type, event.Source ))
	case <- stopCh:
		zap.L().Warn("receive stop channel, stop!!!")
	}
}