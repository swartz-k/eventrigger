package v1

type TargetType string

var (
	MQTTTargetType       TargetType = "mqtt"
	RedisMonitorType     TargetType = "redis"
	KafkaTargetType      TargetType = "kafka"
	HttpTargetType       TargetType = "http"
	CloudEventTargetType TargetType = "cloud_events"
	K8SEventsTargetType  TargetType = "events"
)

// Target common monitor which can produce events to Target K8S resource.
type Target struct {
	// Type is which parse handler to exec
	Type string `json:"type" protobuf:"bytes,1,name=type"`
	// Meta is a unique name of this dependency
	Meta map[string]string `json:"meta" protobuf:"bytes,2,name=meta"`
}

type MQTTTarget struct {
	URL      string `json:"url" yaml:"url" protobuf:"bytes,1,opt,name=url"`
	Topic    string `json:"topic" yaml:"topic" protobuf:"bytes,2,opt,name=topic"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}

type KafkaTarget struct {
	Host     string `json:"host" yaml:"host" protobuf:"bytes,1,opt,name=host"`
	Database string `json:"database" yaml:"database" protobuf:"bytes,2,opt,name=database"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}

type RedisTarget struct {
	Addr     string `json:"addr" yaml:"addr" protobuf:"bytes,1,opt,name=addr"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,2,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,3,opt,name=password"`
	DB       string `json:"db" yaml:"db" protobuf:"bytes,4,opt,name=db"`
	Channel  string `json:"channel" yaml:"channel" protobuf:"bytes,5,opt,name=channel"`
}

type K8sEventsTarget struct {
	Namespace string `json:"namespace" yaml:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	Source    string `json:"source" yaml:"source" protobuf:"bytes,1,opt,name=source"`
	Type      string `json:"type" yaml:"type" protobuf:"bytes,2,opt,name=type"`
}

type CloudEventsTarget struct {
	Source  string `json:"source" yaml:"source" protobuf:"bytes,1,opt,name=source"`
	Type    string `json:"type" yaml:"type" protobuf:"bytes,2,opt,name=type"`
	Version string `json:"version" yaml:"version" protobuf:"bytes,3,opt,name=version"`
}

type HttpTarget struct {
	Hosts   []string `json:"hosts" yaml:"hosts" protobuf:"bytes,1,opt,name=hosts"`
	Headers []string `yaml:"headers" yaml:"headers" protobuf:"bytes,2,opt,name=headers"`
	Suffix  string   `yaml:"suffix" yaml:"suffix" protobuf:"bytes,3,opt,name=suffix"`
}
