package k8s

import (
	"context"
	"eventrigger.com/operator/common/consts"
	commonEvent "eventrigger.com/operator/common/event"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"strconv"

	"eventrigger.com/operator/common/utils/k8s"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"time"
)

var clusterResources = map[string]bool{
	"namespaces": true,
	"nodes":      true,
}

func ScaleObjTo(ctx context.Context, cli *kubernetes.Clientset, obj *unstructured.Unstructured, replicas int32) (err error) {
	namespace := obj.GetNamespace()
	name := obj.GetName()
	switch obj.GetKind() {
	case "statefulset":
		s, err := cli.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "get scale for statefulset %s-%s", namespace, name)
		}
		s.Spec.Replicas = replicas
		_, err = cli.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, s, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "scale statefulset %s-%s to %d", namespace, name, replicas)
		}
		return nil
	case "deployment":
		s, err := cli.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "get scale for deployment %s-%s", namespace, name)
		}
		s.Spec.Replicas = replicas
		_, err = cli.AppsV1().Deployments(namespace).UpdateScale(ctx, name, s, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "scale deployment %s-%s to %d", namespace, name, replicas)
		}
		return nil
	default:
		return errors.New(fmt.Sprintf("not supported %s %s-%s scale", obj.GetKind(), namespace, name))
	}
}

func (r *k8sActor) Exec(ctx context.Context, event commonEvent.Event) error {

	namespace := ""
	if _, isClusterResource := clusterResources[r.GVR.Resource]; !isClusterResource {
		namespace = r.Obj.GetNamespace()
		// Defaults to sensor's namespace
		if namespace == "" {
			namespace = event.Namespace
		}
		if namespace == "" {
			namespace = "default"
		}
	}
	r.Obj.SetNamespace(namespace)
	zap.L().Info("starting operate trigger resource", zap.String("gvr", r.GVR.String()),
		zap.String("op", string(r.OP)), zap.String("namespace", namespace))

	dynamicClient, err := dynamic.NewForConfig(r.Cfg)
	if err != nil {
		return err
	}
	if dynamicClient == nil {
		return errors.New("dynamic client is nil")
	}

	switch r.OP {
	case v1.Create:
		labels := r.Obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		SetEventLabel(event, labels)
		r.Obj.SetLabels(labels)
		_, err = dynamicClient.Resource(r.GVR).Namespace(namespace).Create(ctx, r.Obj, metav1.CreateOptions{})
		if err != nil {
			return errors.Errorf("failed to create object. err: %+v\n", err)
		}
		return nil
	case v1.Delete:
		_, err = dynamicClient.Resource(r.GVR).Namespace(namespace).Get(ctx, r.Obj.GetName(), metav1.GetOptions{})

		if err != nil && apierrors.IsNotFound(err) {
			zap.L().Info("object not found, nothing to delete...")
			return nil
		} else if err != nil {
			return errors.Errorf("failed to retrieve existing object. err: %+v\n", err)
		}

		err = dynamicClient.Resource(r.GVR).Delete(ctx, r.Obj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return errors.Errorf("failed to delete object. err: %+v\n", err)
		}
		return nil
	case v1.ScaleToZero:
		k8sCli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return err
		}
		err = ScaleObjTo(ctx, k8sCli, r.Obj, 0)
		if err != nil {
			return errors.Errorf("failed to scaleToZero. err: %+v\n", err)
		}
		return nil
	case v1.CreateAndScale:
		k8sCli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return err
		}
		labels := r.Obj.GetLabels()
		SetEventLabel(event, labels)
		r.Obj.SetLabels(labels)
		err = ScaleObjTo(ctx, k8sCli, r.Obj, 1)
		if err != nil {
			return errors.Errorf("failed to scaleUp. err: %+v\n", err)
		}
		return nil
	case v1.ScaleUp:
		k8sCli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return err
		}
		labels := r.Obj.GetLabels()
		SetEventLabel(event, labels)
		r.Obj.SetLabels(labels)
		err = ScaleObjTo(ctx, k8sCli, r.Obj, 1)
		if err != nil {
			return errors.Errorf("failed to scaleUp. err: %+v\n", err)
		}
		return nil
	default:
		return errors.Errorf("unknown operation type %s", r.OP)
	}
}

func (r *k8sActor) ScaleToZero(ctx context.Context) error {
	k8sCli, err := kubernetes.NewForConfig(r.Cfg)
	if err != nil {
		return err
	}
	obj, err := k8s.DecodeAndUnstructure(r.Source.Value)
	if err != nil {
		return err
	}
	err = ScaleObjTo(ctx, k8sCli, obj, 0)
	if err != nil {
		return errors.Errorf("failed to scaleToZero. err: %+v\n", err)
	}
	return nil
}

func (r *k8sActor) GetScaleToZeroTime() *time.Duration {
	return r.ScaleToZeroTime
}

func (r *k8sActor) String() string {
	return fmt.Sprintf("%s-%s", r.GVR.String(), r.OP)
}

func SetEventLabel(event commonEvent.Event, labels map[string]string) {
	labels[consts.ActionTimestamp] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
	labels[consts.EventNamespace] = event.Namespace
	labels[consts.EventType] = event.Type
	labels[consts.EventSource] = event.Source
	labels[consts.EventData] = event.Data
	labels[consts.EventVersion] = event.Version
}
