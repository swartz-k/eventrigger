package monitor

import (
	"context"
	"eventrigger.com/operator/common/consts"
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/server"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"strings"
)

type HttpOptions struct {
	Hosts   []string
	Headers map[string]string
	Suffix  string
}

type HttpMonitor struct {
	Opts *HttpOptions

	EventChannel chan event.Event
	StopCh       <-chan struct{}
}

func parseHttpMeta(meta map[string]string) (opts *HttpOptions, err error) {
	opts = &HttpOptions{}

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

func NewHttpMonitor(meta map[string]string) (*HttpMonitor, error) {
	opts, err := parseHttpMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse http meta")
	}

	m := &HttpMonitor{
		Opts: opts,
	}

	return m, nil
}
func (m *HttpMonitor) Handler(c *gin.Context) (int, interface{}, error) {
	// send event to actor
	uuid := c.Request.Header.Get(consts.UUIDLabelHeader)
	var data string
	rawData, err := c.GetRawData()
	if err != nil {
		data = ""
	} else {
		data = string(rawData)
	}
	sEvent := event.NewEvent("", string(v1.HttpMonitorType), "", "", data, uuid)
	m.EventChannel <- sEvent
	return 0, sEvent, nil
}

func (m *HttpMonitor) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {
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
