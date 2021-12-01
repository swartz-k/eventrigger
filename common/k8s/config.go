package k8s

import (
	"eventrigger.com/operator/common/consts"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func GetKubeConfig() (cfg *rest.Config, err error) {
	kubeConfig := os.Getenv(consts.EnvDefaultKubeConfig)

	if kubeConfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	return cfg, err
}
