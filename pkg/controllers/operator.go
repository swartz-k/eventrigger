/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/sync/errsgroup"
	eventriggerv1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/controllers/actor"
	"eventrigger.com/operator/pkg/controllers/events/cloudevents"
	"eventrigger.com/operator/pkg/controllers/events/k8sevent"
	"eventrigger.com/operator/pkg/controllers/events/monitor"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"eventrigger.com/operator/pkg/controllers/sensor"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(eventriggerv1.AddToScheme(scheme))
	utilruntime.Must(eventriggerv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type OperatorOptions struct {
	Port        int
	MetricsPort int
	HealthPort  int
	LeaderElect bool

	// event
	CloudEventsPort uint `json:"cloud_events_port" yaml:"cloud_events_port"`

	// trigger
	EventFrom   string `yaml:"event_from" yaml:"event_from"`     // how to attach event to trigger source, maybe: env,cm,secret
	EventFormat string `json:"event_format" yaml:"event_format"` // which event should be formatted, maybe: json, string, yaml, toml
}

type Operator struct {
	CTX     context.Context
	Options OperatorOptions

	// monitor
	MonitorManager monitor.Manager

	// event monitor
	CloudEventsController event.Controller
	EventsController      event.Controller
	// controller
	Controller *manager.Manager
	// actor
	Actor          *actor.Runner
	Cfg            *rest.Config
	MonitorChannel chan eventriggerv1.Monitor
	EventChannel   chan event.Event
	stopCh         <-chan struct{}
}

func NewOperator(options *OperatorOptions) (op *Operator, err error) {
	//var metricsAddr string
	//var enableLeaderElection bool
	//var probeAddr string
	//flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	//flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	//flag.BoolVar(&enableLeaderElection, "leader-elect", false,
	//	"Enable leader election for controller manager. "+
	//		"Enabling this will ensure there is only one active controller manager.")
	if options == nil {
		options = &OperatorOptions{
			Port:            7080,
			MetricsPort:     7081,
			HealthPort:      7082,
			CloudEventsPort: 7088,
			LeaderElect:     true,
		}
	} else {
		if options.HealthPort == 0 || options.MetricsPort == 0 || options.CloudEventsPort == 0 {
			return nil, errors.New("operator options port should not be 0")
		}
	}
	op = &Operator{
		CTX:            context.Background(),
		Options:        *options,
		EventChannel:   make(chan event.Event, 100),
		MonitorChannel: make(chan eventriggerv1.Monitor, 100),
	}
	op.CloudEventsController, err = cloudevents.NewCloudEventController(
		op.Options.CloudEventsPort)
	if err != nil {
		return nil, errors.Wrap(err, "init cloud events controller")
	}

	op.EventsController, err = k8sevent.NewEventController(op.stopCh)
	if err != nil {
		return nil, errors.Wrap(err, "init kubernetes events controller")
	}

	op.Actor, err = actor.NewRunner(op.CTX, op.EventChannel, op.stopCh)
	if err != nil {
		return nil, errors.Wrap(err, "init kubernetes actor runner")
	}

	op.MonitorManager, err = monitor.NewManager(op.CTX, op.stopCh, op.MonitorChannel)
	if err != nil {
		return nil, errors.Wrap(err, "init monitor manager")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     fmt.Sprintf(":%d", op.Options.MetricsPort),
		Port:                   op.Options.Port,
		HealthProbeBindAddress: fmt.Sprintf(":%d", op.Options.HealthPort),
		LeaderElection:         op.Options.LeaderElect,
		LeaderElectionID:       "7159574d.eventrigger.com",
	})
	if err != nil {
		return nil, errors.Wrap(err, "init manager")
	}

	if err = (&sensor.SensorReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		MonitorChan: op.MonitorChannel,
	}).SetupWithManager(mgr); err != nil {
		return nil, errors.Wrap(err, "unable to create controller Sensor")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, errors.Wrap(err, "unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, errors.Wrap(err, "unable to set up ready check")
	}

	op.Controller = &mgr
	return op, nil
}

func (op *Operator) Run() error {

	var cfg zap.Config = zap.NewProductionConfig()

	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	undo := zap.ReplaceGlobals(logger)
	defer undo()

	var eg errsgroup.Group
	eg.Go(func() error {
		err := op.MonitorManager.Run()
		zap.L().Info("monitor manager", zap.Error(err))
		return err
	})

	eg.Go(func() error {
		err := op.CloudEventsController.Run(op.CTX, op.EventChannel)
		zap.L().Info("cloud events controller", zap.Error(err))
		return err
	})

	eg.Go(func() error {
		err := op.EventsController.Run(op.CTX, op.EventChannel)
		zap.L().Info("k8s events controller", zap.Error(err))
		return err
	})

	eg.Go(func() error {
		err := op.Actor.Run(op.CTX)
		zap.L().Info("actor run", zap.Error(err))
		return err
	})

	return eg.WaitWithStopChannel(op.stopCh)

}
