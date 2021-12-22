package server

import (
	cEvent "eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/k8s"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viney-shih/go-lock"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	GlobalK8sEventsMonitor *k8sEventsMonitor
)

type k8sEventsMonitor struct {
	Cfg       *rest.Config
	ClientSet *kubernetes.Clientset

	InformerFactory      informers.SharedInformerFactory
	EventsLister         listerCoreV1.EventLister
	EventsSynced         cache.InformerSynced
	Workqueue            workqueue.RateLimitingInterface
	EventInformerCacheRW *lock.CASMutex

	EventChannelMapper map[string]*chan cEvent.Event
	StopChan           chan struct{}
}

func NewK8sEventsMonitor() (monitor *k8sEventsMonitor, err error) {
	c := &k8sEventsMonitor{
		Workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "k8sEvents"),
		EventInformerCacheRW: lock.NewCASMutex(),
		EventChannelMapper:   make(map[string]*chan cEvent.Event),
		StopChan:             make(chan struct{}),
	}
	cfg, err := k8s.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	c.Cfg = cfg
	c.ClientSet, err = kubernetes.NewForConfig(c.Cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new for k8s config")
	}

	c.InformerFactory = informers.NewSharedInformerFactory(c.ClientSet, 0)
	informer := c.InformerFactory.Core().V1().Events()
	c.EventsLister = informer.Lister()
	c.EventsSynced = informer.Informer().HasSynced
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event, ok := obj.(*v1.Event)
			if !ok {
				zap.L().Error("failed")
			}
			key := UniqueK8sEventKey(event.Kind, event.Type, event.APIVersion, event.Namespace)
			channel, ok := c.EventChannelMapper[key]
			if ok {
				ce := cEvent.Event{
					Namespace: event.Namespace,
					Source:    event.Source.String(),
					Type:      event.Type,
					Version:   event.APIVersion,
					Data:      event.Message,
				}
				*channel <- ce
			}
		},
	})
	return c, nil
}

func (c *k8sEventsMonitor) Run() error {
	c.InformerFactory.Start(c.StopChan)
	defer runtime.HandleCrash()
	defer c.Workqueue.ShutDown()

	zap.L().Info("Waiting for informer caches to sync")
	if ok := cache.WaitForNamedCacheSync("k8sEvents", c.StopChan, c.EventsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	zap.L().Info("Starting event workers")
	// Launch two workers to process Foo resources
	for i := 0; i < 5; i++ {
		go wait.Until(c.runWorker, time.Second, c.StopChan)
	}

	klog.Info("Started event workers")
	<-c.StopChan
	klog.Info("Shutting down event workers")

	return nil
}

func (c *k8sEventsMonitor) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *k8sEventsMonitor) processNextWorkItem() bool {
	obj, shutdown := c.Workqueue.Get()

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
		defer c.Workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.Workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.Workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.Workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *k8sEventsMonitor) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	_, err = c.EventsLister.Events(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if apierrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("event '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	return nil
}

func UniqueK8sEventKey(eventKind, eventType, eventAPIVersion, namespace string) string {
	return fmt.Sprintf("%s%s-%s-%s", eventKind, eventType, eventAPIVersion, namespace)
}

func (c *k8sEventsMonitor) UpdateMonitor(key string, channel *chan cEvent.Event) {
	c.EventChannelMapper[key] = channel
}

func (c *k8sEventsMonitor) DeleteMonitor(key string) {
	delete(c.EventChannelMapper, key)
}
