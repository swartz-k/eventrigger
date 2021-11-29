package k8s

import (
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/pkg/api/core/common"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type k8sActor struct {
	OP     string
	Source *common.Resource
	Cfg    *rest.Config
}

func (k k8sActor) Run(stopCh <-chan struct{}) error {
	panic("implement me")
}

func NewK8SActor(op string, source *common.Resource) (actor *k8sActor, err error) {
	if source == nil {
		return nil, errors.New("k8s actor resource is nil")
	}
	kubeConfig := os.Getenv(consts.EnvDefaultKubeConfig)

	var cfg *rest.Config
	if kubeConfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	return &k8sActor{OP: op, Source: source, Cfg: cfg}, nil
}
