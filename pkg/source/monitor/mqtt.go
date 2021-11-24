package monitor

import (
	"eventrigger.com/operator/common/event"
	"eventrigger.com/operator/common/sync/errsgroup"
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

func NewMQTTRunner(opts *MQTTOptions) (Runner, error) {
	if opts == nil || opts.URI == "" || opts.Username == "" || opts.Password == "" {
		return nil, errors.New("NewMQTTRunner failed uri username or password is empty")
	}
	if opts.PingTimeoutSecond == 0 {
		opts.PingTimeoutSecond = 1
	}
	m := &MQTTRunner{Opts: *opts}

	return m, nil
}

func MQTTEventHandler(cli mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func (m *MQTTRunner) Run(eventChannel chan event.Event, stopCh <- chan struct{}) error {
	// Subscribe to a topic
	fmt.Printf("mqtt runner subscribe topic %s \n", m.Opts.Topic)
	m.EventChannel = eventChannel
	m.StopCh = stopCh

	clientOpts := mqtt.NewClientOptions().AddBroker(m.Opts.URI).
		SetUsername(m.Opts.Username).SetPassword(m.Opts.Password)

	clientOpts.SetPingTimeout(time.Duration(m.Opts.PingTimeoutSecond) * time.Second)
	clientOpts.SetDefaultPublishHandler(MQTTEventHandler)
	clientOpts.SetOrderMatters(false)
	fmt.Printf("new client topic %s\n", m.Opts.Topic)
	cli := mqtt.NewClient(clientOpts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		zap.L().Fatal("connect", zap.Error(token.Error()))
	}

	var eg errsgroup.Group
	eg.Go(func() error {
		cli := cli
		token := cli.Subscribe(m.Opts.Topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
		})
		if token.Wait() && token.Error() != nil {
			return errors.Wrapf(token.Error(), "failed to subscribe to the topic %s", m.Opts.Topic)
		}
		return nil
	})
	eg.Go(func() error {
		topic := m.Opts.Topic
		timer := time.NewTicker(1 * time.Second)
		for range timer.C {
			zap.L().Info(fmt.Sprintf("mqtt listening on topic %s", topic))
		}
		return nil
	})
	return eg.WaitWithStopChannel(stopCh)
}
