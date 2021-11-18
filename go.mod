module eventrigger.com/operator

go 1.16

require (
	github.com/Azure/go-amqp v0.13.7
	github.com/Shopify/sarama v1.30.0
	github.com/cloudevents/sdk-go/protocol/amqp/v2 v2.6.1
	github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2 v2.6.1
	github.com/cloudevents/sdk-go/v2 v2.6.1
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/panjf2000/ants/v2 v2.4.6
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/viney-shih/go-lock v1.1.1
	go.uber.org/zap v1.19.0
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/code-generator v0.22.4 // indirect
	sigs.k8s.io/controller-runtime v0.10.0
)
