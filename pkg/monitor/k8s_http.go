package monitor

import (
	"bufio"
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	k8s2 "eventrigger.com/operator/common/k8s"
	"eventrigger.com/operator/common/server"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"strings"
)

type K8sHttpOptions struct {
	Hosts   []string
	Headers map[string]string
	Suffix  string
}

type K8sHttpRunner struct {
	Ctx  context.Context
	Opts *K8sHttpOptions
	// endpoint
	MatchLabels       map[string]string
	EndpointNamespace string
	EndpointUrl       string
	EndpointType      string
	//
	EventChannel chan event.Event
	Retry        int
	StopCh       <-chan struct{}
}

func parseK8sHttpMeta(meta map[string]string) (opts *K8sHttpOptions, err error) {
	opts = &K8sHttpOptions{}

	if hosts, ok := meta["hosts"]; ok {
		opts.Hosts = strings.Split(hosts, ",")
	}
	if suffix, ok := meta["suffix"]; ok {
		opts.Suffix = suffix
	}
	if headerStr, ok := meta["headers"]; ok {
		mapstructure.Decode(headerStr, opts.Headers)
	}

	return opts, nil
}

func NewK8sHttpMonitor(meta map[string]string, trigger *v1.Trigger) (*K8sHttpRunner, error) {
	if trigger == nil || trigger.Template == nil || trigger.Template.K8s == nil || trigger.Template.K8s.Source == nil ||
		trigger.Template.K8s.Source.Resource == nil {
		return nil, errors.New("http monitor only support k8s source and cannot be null")
	}
	opts, err := parseK8sHttpMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse http meta")
	}

	obj, err := k8s2.DecodeAndUnstructure(trigger.Template.K8s.Source.Resource.Value)
	if err != nil {
		return nil, err
	}

	m := &K8sHttpRunner{
		Opts:              opts,
		EndpointNamespace: obj.GetNamespace(),
		Retry:             10,
		MatchLabels:       obj.GetLabels(),
	}

	switch obj.GetKind() {
	case "pod":
		m.EndpointUrl = fmt.Sprintf("%s.%s", obj.GetName(), obj.GetNamespace())
		m.EndpointType = "pod"
	case "statefulset":
		m.EndpointType = "statefulset"
	case "deployment":
		m.EndpointType = "deployment"
	default:
		return nil, errors.New(fmt.Sprintf("not support k8s kind %s", obj.GetKind()))
	}

	return m, nil
}

func (m *K8sHttpRunner) Handler(c *gin.Context) {
	// send event to actor
	var data string
	rawData, err := c.GetRawData()
	if err != nil {
		data = ""
	} else {
		data = string(rawData)
	}
	requestUUID := c.Request.Header.Get(consts.UUIDLabelHeader)
	sendEvent := event.NewEvent("", string(v1.HttpMonitorType), "", "", data, requestUUID)
	m.EventChannel <- sendEvent
	// wait for response of pod

	cfg, err := k8s2.GetKubeConfig()
	if err != nil {
		err = errors.Wrap(err, "new k8s config")
		zap.L().Info(err.Error())
		c.Error(err)
		return
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		err = errors.Wrap(err, "new k8s cli with config")
		zap.L().Info(err.Error())
		c.Error(err)
		return
	}

	labelSelector := labels.Set(m.MatchLabels).AsSelector()

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v12.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labelSelector.String()
				return cli.CoreV1().Pods(m.EndpointNamespace).List(m.Ctx, options)
			},
			WatchFunc: func(options v12.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector.String()
				return cli.CoreV1().Pods(m.EndpointNamespace).Watch(m.Ctx, options)
			},
		},
		&corev1.Pod{},
		1,
		cache.Indexers{},
	)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				zap.L().Error("failed get pod within k8s http")
				return
			}
			ioReader, err := cli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(m.Ctx)
			if err != nil {
				zap.L().Error(fmt.Sprintf("get pods %s-%s log failed err %+v", pod.Namespace, pod.Name, err))
				return
			}
			defer ioReader.Close()
			sc := bufio.NewScanner(ioReader)
			c.Writer.Write(sc.Bytes())
		},
	})

}

func (m *K8sHttpRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {
	m.Ctx = ctx
	m.EventChannel = eventChannel
	m.StopCh = stopCh
	for _, host := range m.Opts.Hosts {
		server.GlobalHttpServer.AddOrReplaceHostMap(host, m.Handler)
	}
	for k, v := range m.Opts.Headers {
		header := fmt.Sprintf("%s=%s", k, v)
		server.GlobalHttpServer.AddOrReplaceHostMap(header, m.Handler)
	}
	return nil
}
