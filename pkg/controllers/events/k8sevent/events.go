package k8sevent

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/utils/k8s"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viney-shih/go-lock"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type eventController struct {
	Cfg             *rest.Config
	ClientSet       *kubernetes.Clientset
	InformerFactory informers.SharedInformerFactory
	EventLister     listerCoreV1.EventLister
	EventSynced     cache.InformerSynced

	EventChannel chan common.Event
	Workqueue    workqueue.RateLimitingInterface
	StopChan     <-chan struct{}

	EventInformerCache        map[k8s.CacheKey]informerCoreV1.EventInformer
	EventNamespaceListerCache map[k8s.CacheKey]listerCoreV1.EventNamespaceLister
	EventInformerMuCache      map[k8s.CacheKey]*lock.CASMutex
	EventInformerCacheRW      *lock.CASMutex
}

func NewEventController(stopChan <-chan struct{}) (controller common.Controller, err error) {
	c := &eventController{
		StopChan:                  stopChan,
		Workqueue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "events"),
		EventInformerCache:        map[k8s.CacheKey]informerCoreV1.EventInformer{},
		EventNamespaceListerCache: map[k8s.CacheKey]listerCoreV1.EventNamespaceLister{},
		EventInformerMuCache:      map[k8s.CacheKey]*lock.CASMutex{},
		EventInformerCacheRW:      lock.NewCASMutex(),
	}
	kubeConfig := os.Getenv(consts.EnvDefaultKubeConfig)

	if kubeConfig != "" {
		c.Cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		c.Cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, errors.Wrap(err, "init from config")
	}

	c.ClientSet, err = kubernetes.NewForConfig(c.Cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new for k8s config")
	}

	c.InformerFactory = informers.NewSharedInformerFactory(c.ClientSet, 0)
	informer := c.InformerFactory.Core().V1().Events()
	c.EventLister = informer.Lister()
	c.EventSynced = informer.Informer().HasSynced
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event, ok := obj.(*v1.Event)
			if !ok {
				zap.L().Error("failed")
			}
			zap.L().Info("new event", zap.String("name", event.Name), zap.String("type", event.Type),
				zap.String("source", event.Source.String()))
			ce := common.Event{
				Namespace: event.Namespace,
				Source:    event.Source.String(),
				Type:      event.Type,
				Version:   event.APIVersion,
				Data:      event.Message,
			}
			c.EventChannel <- ce
		},
	})
	return c, nil
}

func (c *eventController) Run(ctx context.Context, eventChannel chan common.Event) error {
	c.EventChannel = eventChannel
	c.InformerFactory.Start(c.StopChan)
	defer runtime.HandleCrash()
	defer c.Workqueue.ShutDown()

	zap.L().Info("Waiting for informer caches to sync")
	if ok := cache.WaitForNamedCacheSync("events", c.StopChan, c.EventSynced); !ok {
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

func (c *eventController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *eventController) processNextWorkItem() bool {
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

func (c *eventController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	_, err = c.EventLister.Events(namespace).Get(name)
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
