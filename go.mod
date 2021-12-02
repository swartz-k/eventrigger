module eventrigger.com/operator

go 1.16

require (
	github.com/Azure/go-amqp v0.13.7
	github.com/Shopify/sarama v1.30.0
	github.com/cloudevents/sdk-go/protocol/amqp/v2 v2.6.1
	github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2 v2.6.1
	github.com/cloudevents/sdk-go/v2 v2.6.1
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/gin-gonic/gin v1.7.7
	github.com/go-redis/redis/v8 v8.11.4
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.1.2
	github.com/mitchellh/mapstructure v1.4.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/panjf2000/ants/v2 v2.4.6
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/viney-shih/go-lock v1.1.1
	go.uber.org/zap v1.19.0
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/code-generator v0.22.4
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/controller-tools v0.7.0
)
