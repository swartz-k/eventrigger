package actor

import (
	"context"
	"eventrigger.com/operator/common/utils/k8s"
	"eventrigger.com/operator/pkg/api/core/common"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	eventscommon "eventrigger.com/operator/pkg/controllers/events/common"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"strconv"
	"time"
)

var clusterResources = map[string]bool{
	"namespaces": true,
	"nodes":      true,
}

func K8STriggerExecute(ctx context.Context, dynamicClient dynamic.Interface, op v1.KubernetesResourceOperation, event eventscommon.Event, resource *common.Resource) (interface{}, error) {
	if resource == nil {
		return nil, errors.New(fmt.Sprintf("failed to interpret the trigger resource, obj %+v", string(resource.Value)))
	}
	obj, err := k8s.DecodeAndUnstructure(resource.Value)
	if err != nil {
		return nil, err
	}

	gvr := k8s.GetGroupVersionResource(obj)
	namespace := ""
	if _, isClusterResource := clusterResources[gvr.Resource]; !isClusterResource {
		namespace = obj.GetNamespace()
		// Defaults to sensor's namespace
		if namespace == "" {
			namespace = event.Namespace
		}
		if namespace == "" {
			namespace = "default"
		}
	}
	obj.SetNamespace(namespace)
	zap.L().Info("starting operate trigger resource", zap.String("gvr", gvr.String()), zap.String("op", string(op)),
		zap.String("namespace", namespace))

	switch op {
	case v1.Create:
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["event.eventrigger.com/action-timestamp"] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
		obj.SetLabels(labels)
		return dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	case v1.Delete:
		_, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, obj.GetName(), metav1.GetOptions{})

		if err != nil && apierrors.IsNotFound(err) {
			zap.L().Info("object not found, nothing to delete...")
			return nil, nil
		} else if err != nil {
			return nil, errors.Errorf("failed to retrieve existing object. err: %+v\n", err)
		}

		err = dynamicClient.Resource(gvr).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return nil, errors.Errorf("failed to delete object. err: %+v\n", err)
		}
		return nil, nil
	default:
		return nil, errors.Errorf("unknown operation type %s", string(op))
	}
}
