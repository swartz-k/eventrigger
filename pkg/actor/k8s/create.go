package k8s

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	"github.com/pkg/errors"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func (r *k8sActor) CreateObj(ctx context.Context, event event.Event, cli dynamic.Interface) (err error) {
	labels := r.Obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	eventDict := GetEventDict(event)
	for k, v := range eventDict {
		labels[k] = v
	}
	r.Obj.SetLabels(labels)
	switch r.Obj.GetKind() {
	case consts.PodKind:
		var pod corev1.Pod
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(r.Obj.UnstructuredContent(), &pod)
		if err != nil {
			return errors.Wrapf(err, "convert obj gvr %s, %s to pod to create", r.GVR, r.Obj.GetName())
		}
		for _, c := range pod.Spec.Containers {
			var eventEnv []corev1.EnvVar
			for k, v := range eventDict {
				eventEnv = append(eventEnv, corev1.EnvVar{Name: k, Value: v})
			}
			c.Env = append(c.Env, eventEnv...)
		}

		cli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return errors.Wrap(err, "get k8s cli for create pod")
		}
		_, err = cli.CoreV1().Pods(pod.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
		return err
	case consts.JobKind:
		var job v1.Job
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(r.Obj.UnstructuredContent(), &job)
		if err != nil {
			return errors.Wrapf(err, "FromUnstructured to statefulset")
		}
		for _, c := range job.Spec.Template.Spec.Containers {
			var eventEnv []corev1.EnvVar
			for k, v := range eventDict {
				eventEnv = append(eventEnv, corev1.EnvVar{Name: k, Value: v})
			}
			c.Env = append(c.Env, eventEnv...)
		}
		cli, err := kubernetes.NewForConfig(r.Cfg)
		if err != nil {
			return errors.Wrap(err, "get k8s cli for create pod")
		}
		_, err = cli.BatchV1().Jobs(job.Namespace).Create(ctx, &job, metav1.CreateOptions{})
		return err
	}
	_, err = cli.Resource(r.GVR).Namespace(r.Obj.GetNamespace()).Create(ctx, r.Obj, metav1.CreateOptions{})
	if err != nil {
		return errors.Errorf("failed to create object. err: %+v\n", err)
	}
	return nil
}
