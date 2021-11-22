package monitor

import (
	"eventrigger.com/operator/common/event"
	v1 "eventrigger.com/operator/pkg/api/core/v1"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"os"
	"time"
)

type MQTTOptions struct {
	URI      string
	Broker   string
	Username string
	Password string
}

type MQTTRunner struct {
	Cli          mqtt.Client
	EventChannel chan event.Event
}

func NewMQTTRunner(monitor *v1.MQTTMonitor) (Runner, error) {
	m := &MQTTRunner{}

	clientOpts := mqtt.NewClientOptions().AddBroker(monitor.URL).
		SetUsername(monitor.Username).SetPassword(monitor.Password)

	clientOpts.SetPingTimeout(1 * time.Second)
	clientOpts.SetDefaultPublishHandler(m.EventHandler)

	m.Cli = mqtt.NewClient(clientOpts)

	return m, nil
}

func (m *MQTTRunner) EventHandler(cli mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func (m *MQTTRunner) Run(eventChannel chan event.Event) error {
	// Subscribe to a topic
	m.EventChannel = eventChannel
	token := m.Cli.Subscribe("testtopic/#", 0, m.EventHandler)
	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	return nil
}
