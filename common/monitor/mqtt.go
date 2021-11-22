package monitor

import (
	"eventrigger.com/operator/common/event"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"net/url"
	"fmt"
	"time"
	"os"
)

type MQTTOptions struct {
	URI string
	Broker string
	Username string
	Password string
}

type MQTTRunner struct {
	Cli mqtt.Client
	EventChannel chan event.Event
}

func NewMQTTRunner(opts *MQTTOptions) (Runner, error) {

	clientOpts := mqtt.NewClientOptions().AddBroker(opts.URI).
		SetClientID(opts.Broker).SetUsername(opts.Username).SetPassword(opts.Username).SetPassword(opts.Password)

	clientOpts.SetPingTimeout(1 * time.Second)

	m := &MQTTRunner{}
	m.Cli = mqtt.NewClient(clientOpts)

	return m, nil
}

func (m *MQTTRunner) EventHandler(cli mqtt.Client, msg mqtt.Message) {
	cli.
}

func (m *MQTTRunner) Run(eventChannel chan event.Event) error {
	// Subscribe to a topic
	m.EventChannel = eventChannel
	token := m.Cli.Subscribe("testtopic/#", 0, m.EventHandler)
	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

}
