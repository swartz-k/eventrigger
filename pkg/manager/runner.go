package manager

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/pkg/actor"
	"eventrigger.com/operator/pkg/actor/k8s"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/target"
	"eventrigger.com/operator/pkg/trigger"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strconv"
	"sync"
	"time"
)

type RunnerInterface interface {
	Run() error
	Stop()
}

type runner struct {
	CTX     context.Context
	eventCh chan event.Event
	stopCh  chan struct{}

	Sensor  *v1.Sensor
	Trigger trigger.Interface
	Actor   actor.Interface
	Target  target.Interface
	// Config
	IdleTime time.Duration

	// Runtime
	EventMutex sync.Mutex
	EventCount int64
	EventLast  time.Time
}

func ParseSensorTrigger(spec *v1.SensorSpec) (source trigger.Interface, err error) {
	if spec == nil || len(spec.Trigger.Meta) == 0 {
		return nil, errors.New(fmt.Sprintf("sensor %+v trigger or meta is nil", spec))
	}
	m := spec.Trigger
	switch m.Type {
	case string(v1.MQTTTriggerType):
		return trigger.NewMQTTMonitor(m.Meta)
	case string(v1.CronTriggerType):
		return trigger.NewCronMonitor(m.Meta)
	case string(v1.KafkaTriggerType):
		return trigger.NewKafkaMonitor(m.Meta)
	case string(v1.RedisMonitorType):
		return trigger.NewRedisMonitor(m.Meta)
	case string(v1.K8sEventsTriggerType):
		return trigger.NewK8sEventsTrigger(m.Meta)
	case string(v1.CloudEventsTriggerType):
		return trigger.NewCloudEventsTrigger(m.Meta)
	case string(v1.K8sHttpTriggerType):
		if spec.Actor.Template == nil {
			return nil, errors.New("trigger cannot be nil while using k8s http monitor")
		}
		if spec.Actor.Template.K8s == nil {
			return nil, errors.New("k8s trigger cannot be nil while using k8s http monitor")
		}
		return trigger.NewK8sHttpTrigger(m.Meta, &spec.Actor)
	default:
		return nil, errors.New(fmt.Sprintf("not support monitor of %s", m.Type))
	}
}

func ParseSensorActor(spec *v1.SensorSpec) (actor actor.Interface, err error) {
	if spec == nil || spec.Actor.Template == nil {
		return nil, errors.New("actor template is nil")
	}
	a := spec.Actor
	if a.Template.K8s != nil {
		if a.Template.K8s.Source == nil {
			return nil, errors.New("init k8s actor failed, source is nil")
		}
		return k8s.NewK8SActor(a.Template.K8s)
	}

	if a.Template.HTTP != nil {
		return nil, errors.New("temp not support")
	}
	return nil, errors.New("no valid template")
}

func ParseSensorTarget(spec *v1.SensorSpec) (tar target.Interface, err error) {
	if spec == nil || len(spec.Target.Meta) == 0 {
		zap.L().Info("sensor does not have target")
		return nil, nil
	}
	m := spec.Target
	switch m.Type {
	case string(v1.HttpTargetType):
		return target.NewHttpTarget(m.Meta)
	case string(v1.K8SEventsTargetType):
		return target.NewK8sEventsTarget(m.Meta)
	default:
		return nil, fmt.Errorf("not support target type %s", m.Type)
	}
}

func NewRunner(sensor *v1.Sensor) (r RunnerInterface, err error) {
	ctx := context.Background()
	if sensor == nil {
		return nil, errors.New("sensor is nil, runner failed")
	}
	tri, err := ParseSensorTrigger(&sensor.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sensor %s/%s trigger", sensor.Name, sensor.Namespace)
	}
	act, err := ParseSensorActor(&sensor.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sensor %s/%s actor", sensor.Name, sensor.Namespace)
	}
	tar, err := ParseSensorTarget(&sensor.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sensor %s/%s target", sensor.Name, sensor.Namespace)
	}

	r = &runner{
		CTX:        ctx,
		Trigger:    tri,
		Actor:      act,
		Target:     tar,
		Sensor:     sensor,
		EventLast:  time.Now(),
		eventCh:    make(chan event.Event, 2),
		stopCh:     make(chan struct{}, 2),
		EventMutex: sync.Mutex{},
	}
	return r, nil
}

func (r *runner) Run() error {
	err := r.Trigger.Run(r.CTX, r.eventCh)
	if err != nil {
		return err
	}

	var scaleTime time.Duration
	if idleEnable, ok := r.Sensor.Labels[consts.ScaleToZeroEnable]; ok || idleEnable == "true" {
		idleTimeStr, _ := r.Sensor.Labels[consts.ScaleToZeroIdleTime]
		duration, _ := strconv.Atoi(idleTimeStr)
		if duration == 0 {
			duration = 60
		}

		zap.L().Info(fmt.Sprintf("runner with ticker check %d second", duration))
		scaleTime = time.Duration(duration) * time.Second
	} else {
		scaleTime = time.Hour * 24 * 365 * 10
	}
	ticker := time.NewTicker(scaleTime)

	for {
		select {
		case event := <-r.eventCh:
			zap.L().Info(fmt.Sprintf("receive event %s-%s, exec actor", event.Type, event.Source))
			r.EventMutex.Lock()
			r.EventCount += 1
			r.EventLast = time.Now()
			r.EventMutex.Unlock()
			err := r.Actor.Exec(r.CTX, event)
			if err != nil {
				err = errors.Wrapf(err, "actor exec with event %s-%s", event.Type, event.Source)
				zap.L().Error("", zap.Error(err))
			} else {
				zap.L().Info(fmt.Sprintf("successfully exec event %s-%s with actor", event.Type, event.Source))
			}
			if r.Target != nil {
				err = r.Target.Exec(r.CTX)
				if err != nil {
					err = errors.Wrapf(err, "target exec")
					zap.L().Error("", zap.Error(err))
				}
			}
		case t := <-ticker.C:
			r.EventMutex.Lock()
			err := r.Actor.Check(r.CTX, scaleTime, r.EventLast)
			r.EventMutex.Unlock()
			if err != nil {
				zap.L().Error(fmt.Sprintf("exec Check failed err %s at %d", err.Error(), t.UnixNano()))
			}
		case <-r.stopCh:
			zap.L().Warn("receive stop channel, stop!!!")
			return nil
		}
	}
}

func (r *runner) Stop() {
	r.stopCh <- struct{}{}
	if r.Trigger == nil {
		zap.L().Warn("runner monitor is nil")
		return
	}

	err := r.Trigger.Stop()

	if err != nil {
		err = errors.Wrapf(err, "stop monitor")
		zap.L().Error(err.Error())
	}
}
