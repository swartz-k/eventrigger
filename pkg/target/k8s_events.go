package target

import (
	"context"
	"eventrigger.com/operator/common/k8s"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	k8seventsv1 "k8s.io/api/events/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sEventsOptions struct {
	Namespace  string `json:"namespace"`
	ApiVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Type       string `json:"type"`
	// source will be set eventrigger spec
	// involvedObject will be set actor
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type K8sEventsTarget struct {
	Opts *K8sEventsOptions
	Cli  *kubernetes.Clientset
}

func parseK8sEventsMeta(meta map[string]string) (opts *K8sEventsOptions, err error) {
	opts = &K8sEventsOptions{}

	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func NewK8sEventsTarget(meta map[string]string) (*K8sEventsTarget, error) {

	opts, err := parseK8sEventsMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse http meta")
	}

	cfg, err := k8s.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	m := &K8sEventsTarget{
		Opts: opts,
		Cli:  cli,
	}

	return m, nil
}

func (k *K8sEventsTarget) Exec(ctx context.Context) error {
	event := &k8seventsv1.Event{
		ObjectMeta: v1.ObjectMeta{
			Name:      "",
			Namespace: k.Opts.Namespace,
		},
	}
	_, err := k.Cli.EventsV1().Events(k.Opts.Namespace).Create(ctx, event, v1.CreateOptions{})
	return err
}
