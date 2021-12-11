package k8s

import (
	"context"
	"encoding/json"
	"eventrigger.com/operator/common/consts"
	commonEvent "eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"strconv"

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

// ScaleObjTo only scale to 0 or increase replicas
func ScaleObjTo(ctx context.Context, cli *kubernetes.Clientset, obj *unstructured.Unstructured, replicas int32) (err error) {
	namespace := obj.GetNamespace()
	name := obj.GetName()
	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}
	patchByte, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	switch obj.GetKind() {
	case consts.StatefulSetKind:
		s, err := cli.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "get scale for statefulset %s-%s", namespace, name)
		}
		if replicas == 0 || s.Spec.Replicas < replicas {
			_, err = cli.AppsV1().StatefulSets(namespace).Patch(ctx, s.Name, types.MergePatchType, patchByte, metav1.PatchOptions{})
			if err != nil {
				return errors.Wrapf(err, "scale statefulset %s-%s to %d", namespace, name, replicas)
			}
		}
		return nil
	case consts.DeploymentKind:
		d, err := cli.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) && replicas == 0 {
			zap.L().Info("deployment not exist scale to zero success.")
			return nil
		}
		if err != nil {
			return errors.Wrapf(err, "get scale for deployment %s-%s", namespace, name)
		}
		if replicas == 0 || d.Spec.Replicas < replicas {
			_, err = cli.AppsV1().Deployments(namespace).Patch(ctx, d.Name, types.MergePatchType, patchByte, metav1.PatchOptions{})
			if err != nil {
				return errors.Wrapf(err, "scale deployment %s-%s to %d", namespace, name, replicas)
			}
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
		return r.CreateObj(ctx, event, dynamicClient)
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
	case v1.Scale:
		// Create if not exist
		// see label whether scaleToZero ScaleToZeroEnable
		k8sCli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return err
		}
		var existObj *unstructured.Unstructured
		// todo: update resource
		existObj, err = dynamicClient.Resource(r.GVR).Namespace(r.Obj.GetNamespace()).Get(ctx, r.Obj.GetName(), metav1.GetOptions{})
		if err != nil {
			zap.L().Info(fmt.Sprintf("Get resource of gvr %s, name %s, err %s", r.GVR.String(), r.Obj.GetName(), err.Error()))
			if apierrors.IsNotFound(err) {
				existObj, err = dynamicClient.Resource(r.GVR).Namespace(r.Obj.GetNamespace()).Create(ctx, r.Obj, metav1.CreateOptions{})
				if err != nil {
					zap.L().Info(fmt.Sprintf("Create resource of gvr %s, name %s, err %s", r.GVR.String(), r.Obj.GetName(), err.Error()))
					return err
				}
			} else {
				return errors.Wrapf(err, "failed scale when get obj")
			}
		}
		// todo: HPA with event
		// todo: HPA with resource/limit
		err = ScaleObjTo(ctx, k8sCli, existObj, 1)
		if err != nil {
			return errors.Errorf("failed to scaleObjTo. err: %+v\n", err)
		}
		return nil
	default:
		return errors.Errorf("unknown operation type %s", r.OP)
	}
}

func (r *k8sActor) Check(ctx context.Context, scaleTime time.Duration, lastEvent time.Time) error {
	now := time.Now()
	if lastEvent.Add(scaleTime).After(now) {
		return nil
	}
	obj, err := k8s.DecodeAndUnstructure(r.Source.Value)
	if err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("resource gvr:%s, name %s enable scale to zero and time meet since last event",
		r.GVR, obj.GetName()))
	k8sCli, err := kubernetes.NewForConfig(r.Cfg)
	if err != nil {
		return err
	}

	err = ScaleObjTo(ctx, k8sCli, obj, 0)
	if err != nil {
		return errors.Errorf("failed to scaleToZero. err: %+v\n", err)
	}
	return nil
}

func (r *k8sActor) String() string {
	return fmt.Sprintf("%s-%s", r.GVR.String(), r.OP)
}

func GetEventDict(event commonEvent.Event) (dict map[string]string) {
	dict = map[string]string{}
	if event.UUID != "" {
		dict[consts.UUIDLabel] = event.UUID
	}
	dict[consts.ActionTimestamp] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
	dict[consts.EventNamespace] = event.Namespace
	dict[consts.EventType] = event.Type
	dict[consts.EventSource] = event.Source
	dict[consts.EventData] = event.Data
	dict[consts.EventVersion] = event.Version
	return dict
}
