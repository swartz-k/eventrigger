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
	"eventrigger.com/operator/common/sync/errsgroup"
	"eventrigger.com/operator/pkg/controllers/actor"
	"eventrigger.com/operator/pkg/controllers/events/cloudevents"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"eventrigger.com/operator/pkg/controllers/events/k8sevent"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "eventrigger.com/operator/pkg/api/v1"
	eventriggerv1 "eventrigger.com/operator/pkg/api/v1"
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
	utilruntime.Must(corev1.AddToScheme(scheme))
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

	// event monitor
	CloudEventsController common.Controller
	EventsController      common.Controller
	// controller
	Controller *manager.Manager
	// actor
	Actor          *actor.Runner
	Cfg            *rest.Config
	MonitorChannel chan common.Monitor
	EventChannel   chan common.Event
	stopCh         <-chan struct{}

	Log logr.Logger
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
		CTX:     context.Background(),
		Options: *options,
		Log:     zap.New(),
	}
	op.CloudEventsController, err = cloudevents.NewCloudEventController(op.Options.CloudEventsPort)
	if err != nil {
		return nil, errors.Wrap(err, "init cloud events controller")
	}

	op.EventsController, err = k8sevent.NewEventController()
	if err != nil {
		return nil, errors.Wrap(err, "init kubernetes events controller")
	}

	op.Actor, err = actor.NewRunner()
	if err != nil {
		return nil, errors.Wrap(err, "init kubernetes actor runner")
	}

	ctrl.SetLogger(op.Log)
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

	var eg errsgroup.Group
	eg.Go(func() error {
		return op.CloudEventsController.Run(op.CTX, op.MonitorChannel)
	})

	eg.Go(func() error {
		return op.EventsController.Run(op.CTX, op.MonitorChannel)
	})

	return eg.WaitWithStopChannel(op.stopCh)

}
