package events

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/utils/k8s"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viney-shih/go-lock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	informerCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type eventController struct {
	Cfg                       *rest.Config
	ClientSet                 *kubernetes.Clientset
	EventInformerCache        map[k8s.CacheKey]informerCoreV1.EventInformer
	EventNamespaceListerCache map[k8s.CacheKey]listerCoreV1.EventNamespaceLister
	EventInformerMuCache      map[k8s.CacheKey]*lock.CASMutex
	EventInformerCacheRW      *lock.CASMutex
}

func NewEventController() (controller Controller, err error) {
	c := &eventController{
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
	return c, nil
}

func (c *eventController) GetEventsInformer(ctx context.Context) (informerCoreV1.EventInformer, listerCoreV1.EventNamespaceLister, error) {

	namespace := "default"

	informer, lister, err := k8s.MakeGetInformer(ctx, c.ClientSet, &k8s.MakeGetInformerOption{
		Namespace:       namespace,
		InformerCacheRW: c.EventInformerCacheRW,
		GetInformerFromCache: func(cacheKey k8s.CacheKey) (interface{}, bool) {
			informer, ok := c.EventInformerCache[cacheKey]
			return informer, ok
		},
		GetListerFromCache: func(cacheKey k8s.CacheKey) (interface{}, bool) {
			lister, ok := c.EventNamespaceListerCache[cacheKey]
			return lister, ok
		},
		SetInformerToCache: func(cacheKey k8s.CacheKey, v interface{}) {
			c.EventInformerCache[cacheKey] = v.(informerCoreV1.EventInformer)
		},
		SetListerToCache: func(cacheKey k8s.CacheKey, v interface{}) {
			c.EventNamespaceListerCache[cacheKey] = v.(listerCoreV1.EventNamespaceLister)
		},
		InformerMuCache: c.EventInformerMuCache,
		InformerGetter: func(factory informers.SharedInformerFactory) interface {
			Informer() cache.SharedIndexInformer
		} {
			return factory.Core().V1().Events()
		},
		ListerGetter: func(informer interface{}) interface{} {
			return informer.(informerCoreV1.EventInformer).Lister().Events(namespace)
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return informer.(informerCoreV1.EventInformer), lister.(listerCoreV1.EventNamespaceLister), nil
}

func (c *eventController) Run(ctx context.Context) error {
	informer, _, err := c.GetEventsInformer(ctx)
	if err != nil {
		return err
	}
	defer runtime.HandleCrash()
	fmt.Println("eventcontroller running")
	fmt.Println("%+v", c.EventInformerCache)
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event, ok := obj.(*v1.Event)
			if !ok {
				fmt.Println("failed")
			}
			fmt.Printf("name: %s, type: %s, source: %s\n", event.Name, event.Type, event.Source)
			fmt.Printf("involved kind: %s, name: %s, version: %s\n", event.InvolvedObject.Kind, event.InvolvedObject.Name, event.InvolvedObject.APIVersion)
			fmt.Printf("involved kind: %s, name: %s, version: %s\n", event.Reason, event.Message)
		},
	})
	return nil
}
