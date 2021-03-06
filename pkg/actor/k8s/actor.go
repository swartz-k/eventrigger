package k8s

import (
	k8s2 "eventrigger.com/operator/common/k8s"
	"eventrigger.com/operator/pkg/api/core/common"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type k8sActor struct {
	OP     v1.KubernetesResourceOperation
	Obj    *unstructured.Unstructured
	GVR    schema.GroupVersionResource
	Source *common.Resource
	Cfg    *rest.Config
}

func NewK8SActor(t *v1.StandardK8SActor) (actor *k8sActor, err error) {
	if t.Source == nil {
		return nil, errors.New("k8s actor resource is nil")
	}

	cfg, err := k8s2.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	obj, err := k8s2.DecodeAndUnstructure(t.Source.Resource.Value)
	if err != nil {
		return nil, err
	}

	gvr := k8s2.GetGroupVersionResource(obj)

	actor = &k8sActor{
		Obj:    obj,
		GVR:    gvr,
		OP:     t.Operation,
		Source: t.Source.Resource,
		Cfg:    cfg,
	}

	// todo: obj reference with sensor version
	// delete old obj if obj updated

	return actor, nil
}
