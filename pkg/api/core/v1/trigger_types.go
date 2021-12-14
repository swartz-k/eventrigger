package v1

type TriggerType string

var (
	MQTTTriggerType        TriggerType = "mqtt"
	RedisTriggerType       TriggerType = "redis"
	CronTriggerType        TriggerType = "cron"
	KafkaTriggerType       TriggerType = "kafka"
	HttpTriggerType        TriggerType = "http"
	K8sHttpTriggerType     TriggerType = "k8s_http"
	CloudEventsTriggerType TriggerType = "cloud_events"
	K8sEventsTriggerType   TriggerType = "k8s_events"
)

// Trigger common monitor which can produce events to trigger K8S resource.
type Trigger struct {
	// Type is which parse handler to exec
	Type string `json:"type" protobuf:"bytes,1,name=type"`
	// Meta is a unique name of this dependency
	Meta map[string]string `json:"meta" protobuf:"bytes,2,name=meta"`
}

type CronTrigger struct {
	Cron string `json:"cron" yaml:"cron" protobuf:"bytes,1,opt,name=cron"`
}

type MQTTTrigger struct {
	URL      string `json:"url" yaml:"url" protobuf:"bytes,1,opt,name=url"`
	Topic    string `json:"topic" yaml:"topic" protobuf:"bytes,2,opt,name=topic"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}

type KafkaTrigger struct {
	Host     string `json:"host" yaml:"host" protobuf:"bytes,1,opt,name=host"`
	Database string `json:"database" yaml:"database" protobuf:"bytes,2,opt,name=database"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}

type RedisTrigger struct {
	Addr     string `json:"addr" yaml:"addr" protobuf:"bytes,1,opt,name=addr"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,2,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,3,opt,name=password"`
	DB       string `json:"db" yaml:"db" protobuf:"bytes,4,opt,name=db"`
	Channel  string `json:"channel" yaml:"channel" protobuf:"bytes,5,opt,name=channel"`
}

type K8sEventsTrigger struct {
	Source string `json:"source" yaml:"source" protobuf:"bytes,1,opt,name=source"`
	Type   string `json:"type" yaml:"type" protobuf:"bytes,2,opt,name=type"`
}

type CloudEventsTrigger struct {
	Source  string `json:"source" yaml:"source" protobuf:"bytes,1,opt,name=source"`
	Type    string `json:"type" yaml:"type" protobuf:"bytes,2,opt,name=type"`
	Version string `json:"version" yaml:"version" protobuf:"bytes,3,opt,name=version"`
}

type HttpTrigger struct {
	Hosts   []string `json:"hosts" yaml:"hosts" protobuf:"bytes,1,opt,name=hosts"`
	Headers []string `yaml:"headers" yaml:"headers" protobuf:"bytes,2,opt,name=headers"`
	Suffix  string   `yaml:"suffix" yaml:"suffix" protobuf:"bytes,3,opt,name=suffix"`
}
