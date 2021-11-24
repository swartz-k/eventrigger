package monitor

import (
	"eventrigger.com/operator/common/event"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"net/url"
	"testing"
	"time"
)


func EventHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func TestMQTTProducer(t *testing.T) {
	host := "127.0.0.1:1883"
	username := "user"
	password := "hYiLCwOnNg"
	clientOpts := mqtt.NewClientOptions().AddBroker(host).
		SetUsername(username).SetPassword(password)

	clientOpts.SetPingTimeout(1 * time.Second)
	clientOpts.SetDefaultPublishHandler(EventHandler)
	cli := mqtt.NewClient(clientOpts)
	token := cli.Connect()

	if token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	// Subscribe to a topic
	if token := cli.Subscribe("testtopic/#", 0, nil); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	// Publish a message
	token = cli.Publish("testtopic/1", 0, false, "Hello World")
	token.Wait()

	time.Sleep(20 * time.Second)
	t.Log("Unscribe topic")
	// Unscribe
	if token := cli.Unsubscribe("testtopic/#"); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	cli.Disconnect(250)
}


func TestMQTTSubscribe(t *testing.T) {
	ura := "mqtt://user:hYiLCwOnNg@127.0.0.1:1883/test"
	uri, err := url.Parse(ura)
	if err != nil {
		t.Fatal(err)
	}

	topic := uri.Path[1:len(uri.Path)]
	if topic == "" {
		topic = "test"
	}
	clientId := "main"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientId)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		t.Fatal(err)
	}
	go func() {
		cli := client
		cli.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
		})
	}()
	timer := time.NewTicker(3 * time.Second)
	for tt := range timer.C {
		t.Logf("listening %s", tt.String())
	}

}

func TestMQTTPublish(t *testing.T) {
	opt := &MQTTOptions{
		Topic: "test/1",
		URI: "127.0.0.1:1883",
		Username: "user",
		Password: "hYiLCwOnNg",
	}
	clientOpts := mqtt.NewClientOptions().AddBroker(opt.URI).
		SetUsername(opt.Username).SetPassword(opt.Password)

	clientOpts.SetPingTimeout(time.Duration(1) * time.Second)
	clientOpts.OnConnectionLost = connectLostHandler
	t.Logf("New cli to publish msg %s", opt.Topic)

	clientOpts.SetPingTimeout(1 * time.Second)
	cli := mqtt.NewClient(clientOpts)

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Publish a message
	for i := 0; i < 3; i++ {
		token := cli.Publish(opt.Topic, 0, false, "Hello World")
		if token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
		time.Sleep(1 * time.Second)
	}
	t.Log("publish topic")
	// Unscribe
	if token := cli.Unsubscribe(opt.Topic); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	cli.Disconnect(250)
}

func TestMQTTRunner(t *testing.T) {
	opt := &MQTTOptions{
		Topic: "#",
		URI: "127.0.0.1:1883",
		Username: "user",
		Password: "hYiLCwOnNg",
	}
	m, err := NewMQTTRunner(opt)
	if err != nil {
		t.Fatal(err)
	}

	clientOpts := mqtt.NewClientOptions().AddBroker(opt.URI).
		SetUsername(opt.Username).SetPassword(opt.Password)
	clientOpts.OnConnectionLost = connectLostHandler
	clientOpts.SetPingTimeout(time.Duration(1) * time.Second)

	stopCh := make(<- chan struct{})
	eventChannel := make(chan event.Event)
	m.Run(eventChannel, stopCh)
}