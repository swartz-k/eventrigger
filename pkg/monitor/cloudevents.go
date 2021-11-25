package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/Azure/go-amqp"
	"github.com/Shopify/sarama"
	ceamqp "github.com/cloudevents/sdk-go/protocol/amqp/v2"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"

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
	// Channel
	EventChannel chan event.Event
}

func NewCloudEventController(port uint) (event.Controller, error) {
	c := &controller{
		Port: port,
	}
	return c, nil
}

func (c *controller) Run(ctx context.Context, eventChannel chan event.Event) error {
	c.CTX = ctx
	c.EventChannel = eventChannel
	p, err := cloudevents.NewHTTP()
	if err != nil {
		return errors.Wrap(err, "failed to create protocol")
	}

	c.Receiver, err = cloudevents.NewHTTPReceiveHandler(c.CTX, p, c.Receive)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", c.Port)
	zap.L().Info("will listen on ", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, c.Receiver); err != nil {
		zap.L().Info("unable to start http server ", zap.Error(err))
		return err
	}
	return nil
}

func (c *controller) Receive(cloudevent cloudevents.Event) {
	// do something with events.
	comEvent := event.Event{
		Source:  cloudevent.Source(),
		Type:    cloudevent.Type(),
		Version: cloudevent.SpecVersion(),
		Data:    string(cloudevent.Data()),
	}
	zap.L().Info("cloud events receive ", zap.Any("event", cloudevent))
	c.EventChannel <- comEvent
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
		zap.L().Info("==== Got CloudEvent\n%+v\n----\n", zap.Any("event", e))
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
		zap.L().Info("==== Got CloudEvent\n%+v\n----\n", zap.Any("event", e))
	})
	if err != nil {
		return errors.Wrap(err, "Kafka receiver")
	}
	return nil
}
