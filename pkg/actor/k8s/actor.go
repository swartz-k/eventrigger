package k8s

import (
	"eventrigger.com/operator/common/consts"
	k8s2 "eventrigger.com/operator/common/k8s"
	"eventrigger.com/operator/pkg/api/core/common"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"strconv"
	"time"
)

type k8sActor struct {
	OP     v1.KubernetesResourceOperation
	Obj    *unstructured.Unstructured
	GVR    schema.GroupVersionResource
	Source *common.Resource
	Cfg    *rest.Config
	// ScaleToZeroTime
	EnableScaleZero bool
	ScaleToZeroTime time.Duration
}

func NewK8SActor(t *v1.StandardK8STrigger) (actor *k8sActor, err error) {
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
	labels := obj.GetLabels()
	if v, ok := labels[consts.ScaleToZeroEnable]; ok && v == "true" {
		actor.EnableScaleZero = true
	}
	if idleSecond, ok := labels[consts.ScaleToZeroIdleTime]; ok {
		second, err := strconv.Atoi(idleSecond)
		if err != nil {
			second = 60
		}
		actor.ScaleToZeroTime = time.Second * time.Duration(second)
	}
	return actor, nil
}
