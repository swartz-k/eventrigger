package v1

type MonitorType string

var (
	MQTTMonitorType  MonitorType = "mqtt"
	RedisMonitorType MonitorType = "redis"
)

// Monitor common monitor which can produce events to trigger K8S resource.
type Monitor struct {
	// Source is a unique name of this dependency
	Name string `json:"name" protobuf:"bytes,1,name=name"`
	// Type is a unique name of this dependency
	Template *MonitorTemplate `json:"template" protobuf:"bytes,2,name=template"`
}

// MonitorTemplate is the template that describes trigger specification.
type MonitorTemplate struct {
	// StandardK8STrigger refers to the trigger designed to create or update a generic Kubernetes resource.
	// +optional
	MQTT *MQTTMonitor `json:"mqtt,omitempty" protobuf:"bytes,1,opt,name=mqtt"`

	Redis *RedisMonitor `json:"redis,omitempty" protobuf:"bytes,2,opt,name=redis"`
	// AWSLambda refers to the trigger designed to invoke AWS Lambda function with with on-the-fly constructable payload.
	// +optional
}

type MQTTMonitor struct {
	URL      string `json:"url" protobuf:"bytes,1,opt,name=url"`
	Topic    string `json:"topic" protobuf:"bytes,2,opt,name=topic"`
	Username string `json:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" protobuf:"bytes,4,opt,name=password"`
}

type RedisMonitor struct {
	Host     string `json:"host" protobuf:"bytes,1,opt,name=host"`
	Database string `json:"database" protobuf:"bytes,2,opt,name=database"`
	Username string `json:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" protobuf:"bytes,4,opt,name=password"`
}