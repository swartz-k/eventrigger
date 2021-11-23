package actor

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"eventrigger.com/operator/pkg/generated/clientset/versioned"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"

	"eventrigger.com/operator/pkg/generated/informers/externalversions"
	listerv1 "eventrigger.com/operator/pkg/generated/listers/core/v1"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"os"

	"k8s.io/client-go/rest"
)

type Runner struct {
	CTX             context.Context
	StopCh          <-chan struct{}
	InformerFactory externalversions.SharedInformerFactory
	//
	EventChannel chan event.Event
	Workqueue    workqueue.RateLimitingInterface

	// eventsType:sourceType
	EventMap     map[string]*v1.Event
	TriggerMap   map[string]*v1.Trigger
	Cfg          *rest.Config
	ClientSet    *versioned.Clientset
	SensorLister listerv1.SensorLister
	SensorSynced cache.InformerSynced
}

func NewRunner(ctx context.Context, eventChannel chan event.Event, stopChan <-chan struct{}) (runner *Runner, err error) {
	r := &Runner{
		CTX:          ctx,
		EventChannel: eventChannel,
		EventMap:     map[string]*v1.Event{},
		TriggerMap:   map[string]*v1.Trigger{},
		Workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Sensors"),
		StopCh:       stopChan,
	}
	kubeConfig := os.Getenv(consts.EnvDefaultKubeConfig)

	if kubeConfig != "" {
		r.Cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		r.Cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, errors.Wrap(err, "init from config")
	}

	r.ClientSet, err = versioned.NewForConfig(r.Cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new for k8s config")
	}

	r.InformerFactory = externalversions.NewSharedInformerFactory(r.ClientSet, time.Second*30)
	informer := r.InformerFactory.Core().V1().Sensors()
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sensor, ok := obj.(*v1.Sensor)
			if !ok {
				fmt.Println("failed")
			}
			r.EventMap[string(sensor.UID)] = &sensor.Spec.Event
			r.TriggerMap[string(sensor.UID)] = &sensor.Spec.Trigger
			fmt.Printf("name: %s, events: %s, triggers: %+v\n", sensor.Name, sensor.Spec.Event, sensor.Spec.Trigger)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldSensor, ok := oldObj.(*v1.Sensor)
			if !ok {
				fmt.Println("failed")
			}

			newSensor, ok := newObj.(*v1.Sensor)
			if !ok {
				fmt.Println("failed")
			}
			r.EventMap[string(newSensor.UID)] = &newSensor.Spec.Event
			r.TriggerMap[string(newSensor.UID)] = &newSensor.Spec.Trigger
			fmt.Printf("name: %s, events: %s, triggers: %+v\n", oldSensor.Name, oldSensor.Spec.Event, oldSensor.Spec.Trigger)
		},
		DeleteFunc: func(obj interface{}) {
			sensor, ok := obj.(*v1.Sensor)
			if !ok {
				fmt.Println("failed")
			}
			delete(r.EventMap, string(sensor.UID))
			delete(r.TriggerMap, string(sensor.UID))
			fmt.Printf("delete sensor %s", sensor.Status.Judgment.EventId)
		},
	})
	r.SensorLister = informer.Lister()
	r.SensorSynced = informer.Informer().HasSynced

	return r, nil
}

func (r *Runner) Run(ctx context.Context) error {
	zap.L().Info("Starting workers")
	r.InformerFactory.Start(r.StopCh)

	defer runtime.HandleCrash()
	defer r.Workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(r.StopCh, r.SensorSynced); !ok {
		err := fmt.Errorf("failed to wait for caches to sync")
		zap.L().Error("", zap.Error(err))
		return err
	}

	zap.L().Info("Started workers")
	for i := 0; i < 5; i++ {
		go wait.Until(r.runWorker, time.Second, r.StopCh)
	}
	go r.ReceiverEvent()

	<-r.StopCh
	zap.L().Info("Shutting down workers")

	return nil
}

func (r *Runner) ReceiverEvent() bool {
	zap.L().Info("actor listening event")
	for event := range r.EventChannel {
		zap.L().Info("actor receive", zap.Any("event", event))
		for key, sensorEvent := range r.EventMap {
			zap.L().Info("actor check eventMap", zap.String("id", key),
				zap.String("type", sensorEvent.Type), zap.String("source", sensorEvent.Source))
			if sensorEvent.Type == event.Type && sensorEvent.Source == event.Source {
				go r.DeployTrigger(key, event)
			}
		}
	}
	return true
}

func (r *Runner) runWorker() {
	for r.processNextWorkItem() {
	}
}

func (r *Runner) processNextWorkItem() bool {
	obj, shutdown := r.Workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer r.Workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			r.Workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := r.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			r.Workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		r.Workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (r *Runner) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	sensor, err := r.SensorLister.Sensors(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if apierrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	template := sensor.Spec.Trigger.Template
	if template == nil {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	return nil
}

func (r *Runner) DeployTrigger(id string, event event.Event) {
	trigger, ok := r.TriggerMap[id]
	if !ok {
		zap.L().Info("cannot get trigger for id %s\n", zap.String("id", id))
		return
	}
	tmpl := trigger.Template
	if tmpl == nil {
		zap.L().Info("trigger template is nil")
		return
	}

	if tmpl.K8s != nil {
		err := r.OperateK8sSource(tmpl.K8s, event)
		zap.L().Info("Operator K8S resource with result", zap.Error(err))
		return
	}
	return
}
