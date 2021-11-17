package k8s

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viney-shih/go-lock"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"time"
)

var (
	informerSyncTimeout = 30 * time.Second
)

type CacheKey string

type MakeGetInformerOption struct {
	Namespace            string
	InformerCacheRW      *lock.CASMutex
	GetInformerFromCache func(cacheKey CacheKey) (interface{}, bool)
	GetListerFromCache   func(cacheKey CacheKey) (interface{}, bool)
	SetInformerToCache   func(cacheKey CacheKey, v interface{})
	SetListerToCache     func(cacheKey CacheKey, v interface{})
	InformerMuCache      map[CacheKey]*lock.CASMutex
	InformerGetter       func(factory informers.SharedInformerFactory) interface {
		Informer() cache.SharedIndexInformer
	}
	ListerGetter func(informer interface{}) interface{}
}

func WaitForCacheSync(stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	err := wait.PollImmediateUntil(500*time.Millisecond,
		func() (bool, error) {
			for _, syncFunc := range cacheSyncs {
				if !syncFunc() {
					return false, nil
				}
			}
			return true, nil
		},
		stopCh)
	return err == nil
}

func MakeGetInformer(ctx context.Context, cli *kubernetes.Clientset, option *MakeGetInformerOption) (interface{}, interface{}, error) {
	var cacheKey CacheKey
	if option.Namespace != "" {
		cacheKey = CacheKey(option.Namespace)
	} else {
		cacheKey = "default"
	}

	option.InformerCacheRW.Lock()
	informer, informerOk := option.GetInformerFromCache(cacheKey)
	lister, _ := option.GetListerFromCache(cacheKey)
	informerMu, informerMuOk := option.InformerMuCache[cacheKey]
	if !informerMuOk {
		informerMu = lock.NewCASMutex()
		option.InformerMuCache[cacheKey] = informerMu
	}
	option.InformerCacheRW.Unlock()

	if !informerOk {
		var err error
		informer, lister, err = func() (interface{}, interface{}, error) {
			if !informerMu.TryLockWithContext(ctx) {
				return nil, nil, errors.New("informer locker is busy")
			}

			defer informerMu.Unlock()

			option.InformerCacheRW.RLock()
			informer, informerOk := option.GetInformerFromCache(cacheKey)
			lister, _ := option.GetListerFromCache(cacheKey)
			option.InformerCacheRW.RUnlock()

			if informerOk {
				return informer, lister, nil
			}

			var factory informers.SharedInformerFactory
			if option.Namespace != "" {
				factory = informers.NewSharedInformerFactoryWithOptions(cli, 0, informers.WithNamespace(option.Namespace))
			} else {
				factory = informers.NewSharedInformerFactoryWithOptions(cli, 0)
			}
			sthInformer := option.InformerGetter(factory)

			stopper := make(chan struct{})
			go factory.Start(stopper)
			defer func() {
				if err != nil {
					close(stopper)
				}
			}()

			syncStopper := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					close(syncStopper)
					return
				case <-time.After(informerSyncTimeout):
					close(syncStopper)
					return
				}
			}()

			informer_ := sthInformer.Informer()
			if !WaitForCacheSync(syncStopper, informer_.HasSynced) {
				err = errors.New(fmt.Sprintf("Timed out waiting for caches to sync informer, namespace: %s", option.Namespace))
				runtime.HandleError(err)
				return nil, nil, err
			}

			lister = option.ListerGetter(sthInformer)

			option.InformerCacheRW.Lock()
			option.SetInformerToCache(cacheKey, sthInformer)
			option.SetListerToCache(cacheKey, lister)
			option.InformerCacheRW.Unlock()

			return sthInformer, lister, nil
		}()

		if err != nil {
			return nil, nil, err
		}
	}

	return informer, lister, nil
}
