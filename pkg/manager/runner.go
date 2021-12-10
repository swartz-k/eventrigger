package manager

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/pkg/actor"
	"eventrigger.com/operator/pkg/actor/k8s"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/monitor"
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

	Sensor *v1.Sensor
	Monitor monitor.Interface
	Actor   actor.Interface
	// Config
	IdleTime time.Duration

	// Runtime
	EventMutex sync.Mutex
	EventCount int64
	EventLast  time.Time
}

func ParseSensorMonitor(spec *v1.SensorSpec) (source monitor.Interface, err error) {
	if spec == nil || len(spec.Monitor.Meta) == 0 {
		return nil, errors.New(fmt.Sprintf("sensor %s monitor %+v or meta is nil",  spec))
	}
	m := spec.Monitor
	switch m.Type {
	case string(v1.MQTTMonitorType):
		return monitor.NewMQTTMonitor(m.Meta)
	case string(v1.CronMonitorType):
		return monitor.NewCronMonitor(m.Meta)
	case string(v1.KafkaMonitorType):
		return monitor.NewKafkaMonitor(m.Meta)
	case string(v1.RedisMonitorType):
		return monitor.NewRedisMonitor(m.Meta)
	case string(v1.HttpMonitorType):
		monitor.NewHttpMonitor(m.Meta)
	case string(v1.K8sHttpMonitorType):
		if spec.Trigger.Template == nil {
			return nil, errors.New("trigger cannot be nil while using k8s http monitor")
		}
		if spec.Trigger.Template.K8s == nil {
			return nil, errors.New("k8s trigger cannot be nil while using k8s http monitor")
		}
		return monitor.NewK8sHttpMonitor(m.Meta, &spec.Trigger)
	default:
		return nil, errors.New(fmt.Sprintf("not support monitor of %s", m.Type))
	}
	return
}

func ParseSensorTrigger(spec *v1.SensorSpec) (actor actor.Interface, err error) {
	if spec == nil || spec.Trigger.Template == nil {
		return nil, errors.New("trigger template is nil")
	}
	trigger := spec.Trigger
	if trigger.Template.K8s != nil {
		if trigger.Template.K8s.Source == nil {
			return nil, errors.New("init k8s actor failed, source is nil")
		}
		return k8s.NewK8SActor(trigger.Template.K8s)
	}

	if trigger.Template.HTTP != nil {
		return nil, errors.New("temp not support")
	}
	return nil, errors.New("no valid template")
}

func NewRunner(sensor *v1.Sensor) (r RunnerInterface, err error) {
	ctx := context.Background()
	if sensor == nil {
		return nil, errors.New("sensor is nil, runner failed")
	}
	monitor, err := ParseSensorMonitor(&sensor.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sensor %s/%s monitor", sensor.Name, sensor.Namespace)
	}
	actor, err := ParseSensorTrigger(&sensor.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sensor %s/%s trigger",  sensor.Name, sensor.Namespace)
	}

	r = &runner{
		CTX:     ctx,
		Monitor: monitor,
		Actor:   actor,
		Sensor: sensor,
		EventLast: time.Now(),
		eventCh: make(chan event.Event),
		stopCh: make(chan struct{}, 2),
		EventMutex: sync.Mutex{},
	}
	return r, nil
}

func (r *runner) Run() error {
	err := r.Monitor.Run(r.CTX, r.eventCh)
	if err != nil {
		return err
	}

	var scaleTime time.Duration
	if idleEnable, ok := r.Sensor.Labels[consts.ScaleToZeroEnable]; ok || idleEnable == "true" {
		idleTimeStr, _ := r.Sensor.Labels[consts.ScaleToZeroIdleTime]
		duration, _ := strconv.Atoi(idleTimeStr)
		if duration != 0 {
			duration = 60
		}

		zap.L().Info(fmt.Sprintf("runner with ticker check %d second", duration))
		scaleTime = time.Duration(duration) * time.Second
	} else {
		scaleTime = time.Hour * 24 *365 * 10
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
		case t := <-ticker.C:
			r.EventMutex.Lock()
			err := r.Actor.Check(r.CTX, scaleTime, r.EventLast)
			r.EventMutex.Unlock()
			if err != nil {
				zap.L().Error(fmt.Sprintf("exec Check failed err %s at %d", err.Error(), t.UnixNano()))
			}
		case <- r.stopCh:
			zap.L().Warn("receive stop channel, stop!!!")
			return nil
		}
	}
}

func (r *runner) Stop() {
	r.stopCh <- struct{}{}
	if r.Monitor == nil {
		zap.L().Warn("runner monitor is nil")
		return
	}

	err := r.Monitor.Stop()

	if err != nil {
		err = errors.Wrapf(err, "stop monitor")
		zap.L().Error(err.Error())
	}
}
