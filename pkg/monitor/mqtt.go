package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"time"
)

type MQTTOptions struct {
	URI      string
	Topic    string
	Username string
	Password string

	// Opts
	PingTimeoutSecond int32
}

type MQTTRunner struct {
	Opts   MQTTOptions
	StopCh <-chan struct{}
}

func parseMQTTMeta(meta map[string]string) (*MQTTOptions, error) {
	opts := &MQTTOptions{}
	err := mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse mqtt meta failed %s", meta))
	}

	if opts == nil || opts.URI == "" || opts.Username == "" || opts.Password == "" {
		return nil, errors.New("NewMQTTRunner failed uri username or password is empty")
	}
	if opts.PingTimeoutSecond == 0 {
		opts.PingTimeoutSecond = 1
	}
	return opts, nil
}

func NewMQTTMonitor(meta map[string]string) (*MQTTRunner, error) {
	opts, err := parseMQTTMeta(meta)
	if err != nil {
		return nil, errors.Wrapf(err, "parse mqtt meta")
	}
	m := &MQTTRunner{Opts: *opts}
	return m, nil
}

func (m *MQTTRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {
	// Subscribe to a topic
	m.StopCh = stopCh

	clientOpts := mqtt.NewClientOptions().AddBroker(m.Opts.URI).
		SetUsername(m.Opts.Username).SetPassword(m.Opts.Password)

	clientOpts.SetPingTimeout(time.Duration(m.Opts.PingTimeoutSecond) * time.Second)
	clientOpts.SetOrderMatters(false)

	// todo: retry
	cli := mqtt.NewClient(clientOpts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		err := errors.Wrapf(token.Error(), "connect to mqtt://%s:%s@%s/%s",
			m.Opts.Username, m.Opts.Password, m.Opts.URI, m.Opts.Topic)
		zap.L().Error("connect", zap.Error(err))
		return err
	}

	token := cli.Subscribe(m.Opts.Topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		ev := event.Event{
			Source: msg.Topic(),
			Type:   "mqtt",
			Data:   string(msg.Payload()),
		}
		eventChannel <- ev
	})
	if token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "failed to subscribe to the topic %s", m.Opts.Topic)
	}
	return nil
}
