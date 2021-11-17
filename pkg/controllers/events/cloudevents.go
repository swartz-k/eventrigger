package events

import (
	"context"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"net/http"
)

type cloudEventController struct {
	Port     uint
	Receiver *client.EventReceiver
}

func receive(event cloudevents.Event) {
	// do something with events.
	fmt.Printf("%s", event)
}

func NewCloudEventController(port uint) (Controller, error) {
	ctx := context.Background()
	p, err := cloudevents.NewHTTP()
	if err != nil {
		fmt.Printf("failed to create protocol: %s", err.Error())
	}

	h, err := cloudevents.NewHTTPReceiveHandler(ctx, p, receive)
	if err != nil {
		fmt.Printf("failed to create handler: %s", err.Error())
	}

	c := &cloudEventController{
		Port:     port,
		Receiver: h,
	}
	return c, nil
}

func (c *cloudEventController) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", c.Port)
	fmt.Printf("will listen on %s", addr)
	if err := http.ListenAndServe(addr, c.Receiver); err != nil {
		fmt.Printf("unable to start http server, %s", err)
		return err
	}
	return nil
}
