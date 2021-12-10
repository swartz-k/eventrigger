package monitor

import (
	"bytes"
	"context"
	"encoding/base64"
	json2 "encoding/json"
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
	"io/ioutil"
	v13 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

type K8sHttpOptions struct {
	Hosts   []string
	Headers map[string]string
	Suffix  string
}

type K8sHttpMonitor struct {
	Ctx  context.Context
	Opts *K8sHttpOptions
	// endpoint
	Operation         v1.KubernetesResourceOperation
	MatchLabels       map[string]string
	EndpointNamespace string
	EndpointUrl       string
	EndpointType      string
	//
	EventChannel chan event.Event
	Retry        int
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

func NewK8sHttpMonitor(meta map[string]string, trigger *v1.Trigger) (*K8sHttpMonitor, error) {
	if trigger == nil || trigger.Template == nil || trigger.Template.K8s == nil || trigger.Template.K8s.Source == nil ||
		trigger.Template.K8s.Source.Resource == nil {
		return nil, errors.New("http monitor only support k8s source and cannot be null")
	}
	op := trigger.Template.K8s.Operation
	switch op {
	case v1.Scale:
		break
	case v1.Create:
		break
	default:
		return nil, errors.New(fmt.Sprintf("not support operation %s", op))
	}
	opts, err := parseK8sHttpMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse http meta")
	}
	resource := trigger.Template.K8s.Source.Resource
	obj, err := k8s2.DecodeAndUnstructure(resource.Value)
	if err != nil {
		return nil, err
	}

	m := &K8sHttpMonitor{
		Opts:              opts,
		EndpointNamespace: obj.GetNamespace(),
		Retry:             10,
		Operation:         op,
	}

	switch obj.GetKind() {
	case consts.PodKind:
		m.MatchLabels = obj.GetLabels()
		m.EndpointUrl = fmt.Sprintf("%s.%s", obj.GetName(), obj.GetNamespace())
		m.EndpointType = consts.PodKind
	case consts.StatefulSetKind:
		var sts v13.StatefulSet
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &sts)
		if err != nil {
			return nil, errors.Wrapf(err, "FromUnstructured to statefulset")
		}
		m.MatchLabels = sts.Spec.Selector.MatchLabels
		m.EndpointType = consts.StatefulSetKind
	case consts.DeploymentKind:
		var dm v13.Deployment
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &dm)
		if err != nil {
			return nil, errors.Wrapf(err, "FromUnstructured to statefulset")
		}
		m.MatchLabels = dm.Spec.Selector.MatchLabels
		m.EndpointType = consts.DeploymentKind
	default:
		return nil, errors.New(fmt.Sprintf("not support k8s kind %s", obj.GetKind()))
	}

	return m, nil
}

func (m *K8sHttpMonitor) Handler(c *gin.Context) (code int, resp interface{}, err error) {
	// send event to actor
	var data string
	rawData, err := c.GetRawData()
	if err != nil {
		data = ""
	} else {
		data = string(rawData)
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(rawData))

	requestUUID := c.Request.Header.Get(consts.UUIDLabelHeader)
	sendEvent := event.NewEvent("", string(v1.HttpMonitorType), "", "", data, requestUUID)
	m.EventChannel <- sendEvent

	eventData, err := json2.Marshal(sendEvent)
	if err != nil {
		return 0, nil, err
	}
	b64Event := base64.StdEncoding.EncodeToString(eventData)
	endpoint, err := m.GetEndpointUrl()
	if endpoint != "" && err == nil {
		director := func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = endpoint
			req.URL.Path = c.Request.URL.Path
			// auto add event content
			// todo: maybe set header、query or post form
			req.Header.Add(consts.EVENTB64Header, b64Event)
		}

		proxy := &httputil.ReverseProxy{
			Director: director,
		}
		proxy.ModifyResponse = func(response *http.Response) error {
			fmt.Printf("response %s \n", response.Header)
			return nil
		}

		proxy.ErrorHandler = func (rw http.ResponseWriter, req *http.Request, err error) {
			zap.L().Info(fmt.Sprintf("http: proxy error: %v", err))
			rw.WriteHeader(http.StatusBadGateway)
			resp := server.HttpResponse{Code: 500, Data: err.Error()}
			raw, mErr := json2.Marshal(resp)
			if mErr != nil {
				rw.Write([]byte(errors.Wrap(err, mErr.Error()).Error()))
			} else {
				rw.Write(raw)
			}
			return
		}
		zap.L().Info(fmt.Sprintf("redirect req to host %s, path %s, headers %s", endpoint, c.Request.URL.Path, c.Request.Header))
		// todo: return serve http response to response
		proxy.ServeHTTP(c.Writer, c.Request)

		return
	} else {
		msg := fmt.Sprintf("failed to find endpoint with err %s", err)
		zap.L().Info(msg)
		return 500, msg, err
	}

	return 0, "success", nil
}

func (m *K8sHttpMonitor) GetEndpointUrl() (string, error) {
	cfg, err := k8s2.GetKubeConfig()
	if err != nil {
		err = errors.Wrap(err, "get kube config for get endpoint url")
		return "", err
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		err = errors.Wrap(err, "new k8s cli with config")
		return "", err
	}

	labelSelector := labels.Set(m.MatchLabels).AsSelector()
	watchOptions := v12.ListOptions{LabelSelector: labelSelector.String()}

	timeoutTicker := time.NewTicker(5 * time.Minute)

	switch m.EndpointType {
	case consts.PodKind, consts.StatefulSetKind, consts.DeploymentKind:
		watcher, err := cli.CoreV1().Pods(m.EndpointNamespace).Watch(m.Ctx, watchOptions)
		if err != nil {
			err = errors.Wrapf(err, "k8s watcher pod with labels %s", labelSelector.String())
			return "", err
		}
		defer watcher.Stop()
		for {
			select {
			case event := <-watcher.ResultChan():
				p, ok := event.Object.(*corev1.Pod)
				if !ok {
					break
				}
				if len(p.Status.ContainerStatuses) == 0 {
					break
				}
				ready := true
				for _, sta := range p.Status.ContainerStatuses {
					if !sta.Ready {
						ready = false
					}
				}
				if ready {
					var port int32
					for _, c := range p.Spec.Containers {
						for _, pp := range c.Ports {
							if port == 0 {
								port = pp.ContainerPort
							}
							if p.Name == "http" {
								port = pp.ContainerPort
							}
						}
					}
					if p.Status.PodIP == "" {
						zap.L().Error(fmt.Sprintf("faile to get podIP for pod %s/%s", p.Name, p.Namespace))
					}
					return fmt.Sprintf("%s:%d", p.Status.PodIP, port), nil
				}
			case <-timeoutTicker.C:
				return "", errors.New(fmt.Sprintf("waiting pod %s, %s select %s to be ready timeout", m.EndpointType, m.EndpointNamespace, m.MatchLabels))
			}
		}
	default:
		return "", errors.New(fmt.Sprintf("endpoint type %s not support", m.EndpointType))
	}
	return "", errors.New("cannot find right endpoint")
}

func (m *K8sHttpMonitor) Run(ctx context.Context, eventChannel chan event.Event) error {
	m.Ctx = ctx
	m.EventChannel = eventChannel
	for _, host := range m.Opts.Hosts {
		zap.L().Info(fmt.Sprintf("k8s http monitor add host: %s for %s", host, m.EndpointType))
		server.GlobalHttpServer.AddOrReplaceHostMap(host, m.Handler)
	}
	for k, v := range m.Opts.Headers {
		header := fmt.Sprintf("%s=%s", k, v)
		zap.L().Info(fmt.Sprintf("k8s http monitor add header: %s for %s", header, m.EndpointType))
		server.GlobalHttpServer.AddOrReplaceHostMap(header, m.Handler)
	}

	zap.L().Debug(fmt.Sprintf("k8s http monitor with hosts: %s exist", m.Opts.Hosts))
	return nil
}


func (m *K8sHttpMonitor) Stop() error {
	for _, host := range m.Opts.Hosts {
		server.GlobalHttpServer.DeleteHostMap(host)
	}
	for k, v := range m.Opts.Headers {
		header := fmt.Sprintf("%s=%s", k, v)
		server.GlobalHttpServer.DeleteHeaderMap(header)
	}
	return nil
}
