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

package manager

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/server"
	"eventrigger.com/operator/common/sync/errsgroup"
	"eventrigger.com/operator/pkg/generated/clientset/versioned"
	"eventrigger.com/operator/pkg/generated/informers/externalversions"
	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sync"

	"time"

	eventriggerv1 "eventrigger.com/operator/pkg/api/core/v1"
	listerv1 "eventrigger.com/operator/pkg/generated/listers/core/v1"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"eventrigger.com/operator/pkg/controllers/sensor"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
	Debug       bool
	// event
	CloudEventsPort uint `json:"cloud_events_port" yaml:"cloud_events_port"`

	// trigger
	EventFrom   string `yaml:"event_from" yaml:"event_from"`     // how to attach event to trigger source, maybe: env,cm,secret
	EventFormat string `json:"event_format" yaml:"event_format"` // which event should be formatted, maybe: json, string, yaml, toml
}

type Operator struct {
	CTX     context.Context
	Options OperatorOptions

	// map
	RunnerChannelMap map[string]RunnerInterface

	// controller
	ErrorGroup errsgroup.Group
	WaitGroup  sync.WaitGroup
	Controller *manager.Manager

	InformerFactory externalversions.SharedInformerFactory
	Workqueue       workqueue.RateLimitingInterface
	DeWorkqueue     workqueue.RateLimitingInterface
	Cfg             *rest.Config
	ClientSet       *versioned.Clientset
	SensorLister    listerv1.SensorLister
	SensorSynced    cache.InformerSynced

	stopCh <-chan struct{}
}

func NewOperator(options *OperatorOptions) (op *Operator, err error) {
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
		CTX:              context.Background(),
		Options:          *options,
		RunnerChannelMap: make(map[string]RunnerInterface),
		Workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), consts.SensorName),
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
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
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

	kubeConfig := os.Getenv(consts.EnvDefaultKubeConfig)

	if kubeConfig != "" {
		op.Cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		op.Cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, errors.Wrap(err, "init from config")
	}

	op.ClientSet, err = versioned.NewForConfig(op.Cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new for k8s config")
	}

	op.InformerFactory = externalversions.NewSharedInformerFactory(op.ClientSet, time.Second*30)
	sensorInformer := op.InformerFactory.Core().V1().Sensors()
	sensorInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    op.AddSensorHandler,
		UpdateFunc: op.UpdateSensorHandler,
		DeleteFunc: op.DeleteSensorHandler,
	})
	op.SensorLister = sensorInformer.Lister()
	op.SensorSynced = sensorInformer.Informer().HasSynced

	if err != nil {
		return nil, errors.Wrap(err, "init global http server failed")
	}
	return op, nil
}

func (op *Operator) AddSensorHandler(obj interface{}) {
	zap.L().Debug("add sensor handler receive obj")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	zap.L().Info(fmt.Sprintf("start runner with key %s", key))
	var object *eventriggerv1.Sensor
	var ok bool
	if object, ok = obj.(*eventriggerv1.Sensor); !ok {
		zap.L().Error("decode k8s unstructured")
		return
	}

	r, err := NewRunner(object)
	if err != nil {
		err = errors.Wrap(err, "init runner")
		zap.L().Error(err.Error())
		return
	}
	op.RunnerChannelMap[key] = r
	op.ErrorGroup.Go(func() error {
		defer delete(op.RunnerChannelMap, key)
		err = r.Run()
		if err != nil {
			err = errors.Wrap(err, "run runner")
			zap.L().Error(err.Error())
			return err
		}
		zap.L().Info(fmt.Sprintf("runner with key %s done", key))
		return nil
	})
}

func (op *Operator) UpdateSensorHandler(oldObj, newObj interface{}) {
	var newObject *eventriggerv1.Sensor
	var oldObject *eventriggerv1.Sensor
	var ok bool
	if oldObject, ok = oldObj.(*eventriggerv1.Sensor); !ok {
		zap.L().Error("decode old k8s unstructured")
		return
	}
	if newObject, ok = newObj.(*eventriggerv1.Sensor); !ok {
		zap.L().Error("decode new k8s unstructured")
		return
	}
	if cmp.Equal(oldObject, newObject) {
		zap.L().Debug("old obj and new obj spec is same")
		return
	}
	zap.L().Info(fmt.Sprintf("update handler update obj %s/%s ", newObject.Name, newObject.Namespace))
	op.DeleteSensorHandler(oldObj)
	op.AddSensorHandler(newObj)
}

func (op *Operator) DeleteSensorHandler(obj interface{}) {
	zap.L().Debug("delete sensor handler receive obj")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	zap.L().Info(fmt.Sprintf("delete runner with key %s", key))
	r, ok := op.RunnerChannelMap[key]
	if ok {
		r.Stop()
		return
	} else {
		zap.L().Error(fmt.Sprintf("cannot get runner with key %s", key))
	}
	delete(op.RunnerChannelMap, key)
}

func (op *Operator) Run() error {

	var cfg zap.Config = zap.NewProductionConfig()

	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	if op.Options.Debug {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	undo := zap.ReplaceGlobals(logger)
	defer undo()

	/* global server resource
	GlobalHttpServer every k8s_http request will proxy
	GlobalCloudEventsServer receive cloud events and filter event
	GlobalK8sEventsMonitor listener filter
	*/
	op.ErrorGroup.Go(func() error {
		return server.GlobalHttpServer.Run(fmt.Sprintf(":%d", op.Options.Port))
	})
	op.ErrorGroup.Go(func() error {
		return server.GlobalCloudEventsServer.Run(fmt.Sprintf(":%d", op.Options.CloudEventsPort))
	})
	op.ErrorGroup.Go(func() error {
		server.GlobalK8sEventsMonitor, err = server.NewK8sEventsMonitor()
		if err != nil {
			return errors.Wrap(err, "cannot parse k8s events monitor")
		}
		return server.GlobalK8sEventsMonitor.Run()
	})

	zap.L().Info("Starting workers")
	op.InformerFactory.Start(op.stopCh)

	defer utilruntime.HandleCrash()
	defer op.Workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(op.stopCh, op.SensorSynced); !ok {
		err := fmt.Errorf("failed to wait for caches to sync")
		zap.L().Error("", zap.Error(err))
		return err
	}

	err = op.ErrorGroup.WaitWithStopChannel(op.stopCh)
	if err != nil {
		zap.L().Error("wait with stop channel", zap.Error(err))
	}
	t := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-t.C:
			msg := ""
			for key, _ := range op.RunnerChannelMap {
				msg += fmt.Sprintf("%s,", key)
			}
			zap.L().Debug("runnerMap with listening %s", zap.String("msg", msg))
		case <-op.stopCh:
			zap.L().Info("done")
			return nil
		}
	}
}
