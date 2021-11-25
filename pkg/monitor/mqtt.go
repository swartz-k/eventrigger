package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	Opts         MQTTOptions
	EventChannel chan event.Event
	StopCh <- chan struct{}
}

func NewMQTTMonitor(opts *MQTTOptions) (*MQTTRunner, error) {
	if opts == nil || opts.URI == "" || opts.Username == "" || opts.Password == "" {
		return nil, errors.New("NewMQTTRunner failed uri username or password is empty")
	}
	if opts.PingTimeoutSecond == 0 {
		opts.PingTimeoutSecond = 1
	}
	m := &MQTTRunner{Opts: *opts, EventChannel: make(chan event.Event)}

	return m, nil
}

func MQTTEventHandler(cli mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func (m *MQTTRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <- chan struct{}) error {
	// Subscribe to a topic
	m.StopCh = stopCh

	clientOpts := mqtt.NewClientOptions().AddBroker(m.Opts.URI).
		SetUsername(m.Opts.Username).SetPassword(m.Opts.Password)

	clientOpts.SetPingTimeout(time.Duration(m.Opts.PingTimeoutSecond) * time.Second)
	clientOpts.SetDefaultPublishHandler(MQTTEventHandler)
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
			Source:  msg.Topic(),
			Type: "mqtt",
			Data: string(msg.Payload()),
		}
		eventChannel <- ev
	})
	if token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "failed to subscribe to the topic %s", m.Opts.Topic)
	}
	return nil

}
