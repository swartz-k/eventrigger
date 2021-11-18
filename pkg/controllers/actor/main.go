package actor

import (
	"eventrigger.com/operator/common/utils/k8s"
	"github.com/viney-shih/go-lock"
	informerCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
)

type Runner struct {
	Cfg                       *rest.Config
	ClientSet                 *kubernetes.Clientset
	MonitorNamespaces         map[string]string
	EventInformerCache        map[k8s.CacheKey]informerCoreV1.EventInformer
	EventNamespaceListerCache map[k8s.CacheKey]listerCoreV1.EventNamespaceLister
	EventInformerMuCache      map[k8s.CacheKey]*lock.CASMutex
	EventInformerCacheRW      *lock.CASMutex
}

func NewRunner() (runner *Runner, err error) {
	return
}

func (r *Runner) DeployTrigger() {

}
