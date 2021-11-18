package cloudevents

import (
	"context"
	"eventrigger.com/operator/pkg/controllers/events/common"
	"fmt"
	"github.com/Azure/go-amqp"
	"github.com/Shopify/sarama"
	ceamqp "github.com/cloudevents/sdk-go/protocol/amqp/v2"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"net/http"
)

type Monitor struct {
}

type controller struct {
	CTX      context.Context
	Port     uint
	Receiver *client.EventReceiver
	Monitor  map[string]Monitor
	Log      logr.Logger
	// Channel
	MonitorChannel chan common.Monitor
}

func NewCloudEventController(port uint) (common.Controller, error) {
	c := &controller{
		Port: port,
	}
	return c, nil
}

func (c *controller) Run(ctx context.Context, monitorChannel chan common.Monitor) error {
	c.CTX = ctx
	c.MonitorChannel = monitorChannel
	p, err := cloudevents.NewHTTP()
	if err != nil {
		fmt.Printf("failed to create protocol: %s", err.Error())
	}

	c.Receiver, err = cloudevents.NewHTTPReceiveHandler(c.CTX, p, c.Receive)
	if err != nil {
		fmt.Printf("failed to create handler: %s", err.Error())
	}

	addr := fmt.Sprintf(":%d", c.Port)
	fmt.Printf("will listen on %s", addr)
	if err := http.ListenAndServe(addr, c.Receiver); err != nil {
		fmt.Printf("unable to start http server, %s", err)
		return err
	}

	go c.ReceiveMonitor()

	return nil
}

func (c *controller) Receive(event cloudevents.Event) {
	// do something with events.
	fmt.Printf("%s", event)
	c.Log.V(1).Info("%+v", event)
}

func (c *controller) ReceiveMonitor() {
	// do something with events.
	for monitor := range c.MonitorChannel {
		fmt.Printf("%+v", monitor)
	}
}

// NewMQTTMonitor new MQTT monitor
func (c *controller) NewMQTTMonitor(host, node string, opts []ceamqp.Option) error {
	p, err := ceamqp.NewProtocol(host, node, []amqp.ConnOption{}, []amqp.SessionOption{}, opts...)
	if err != nil {
		return errors.Wrap(err, "create AMQP protocol")
	}

	// Close the connection when finished
	defer p.Close(context.Background())

	// Create a new client from the given protocol
	cli, err := cloudevents.NewClient(p)
	if err != nil {
		return errors.Wrap(err, "create client")
	}
	err = cli.StartReceiver(context.Background(), func(e cloudevents.Event) {
		c.Log.V(1).Info("==== Got CloudEvent\n%+v\n----\n", e)
	})

	if err != nil {
		return errors.Wrap(err, "start receiver")
	}
	return nil
}

// NewKafkaMonitor create a kafka client monitor
func (c *controller) NewKafkaMonitor(brokers []string, group, topic string) error {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_0_0_0

	receiver, err := kafka_sarama.NewConsumer(brokers, saramaConfig, group, topic)
	if err != nil {
		return errors.Wrap(err, "create kafka protocol")
	}

	defer receiver.Close(context.Background())

	// Create a new client from the given protocol
	cli, err := cloudevents.NewClient(receiver)
	if err != nil {
		return errors.Wrap(err, "create client")
	}
	err = cli.StartReceiver(context.Background(), func(e cloudevents.Event) {
		c.Log.V(1).Info("==== Got CloudEvent\n%+v\n----\n", e)
	})
	if err != nil {
		return errors.Wrap(err, "Kafka receiver")
	}
	return nil
}
