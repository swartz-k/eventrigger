package v1

type MonitorType string

var (
	MQTTMonitorType  MonitorType = "mqtt"
	RedisMonitorType MonitorType = "redis"
	CronMonitorType  MonitorType = "cron"
	KafkaMonitorType MonitorType = "kafka"
)

// Monitor common monitor which can produce events to trigger K8S resource.
type Monitor struct {
	// Type is which parse handler to exec
	Type string `json:"type" protobuf:"bytes,1,name=type"`
	// Meta is a unique name of this dependency
	Meta map[string]string `json:"meta" protobuf:"bytes,2,name=meta"`
}

type CronMonitor struct {
	Cron string `json:"cron" yaml:"cron" protobuf:"bytes,1,opt,name=cron"`
}

type MQTTMonitor struct {
	URL      string `json:"url" yaml:"url" protobuf:"bytes,1,opt,name=url"`
	Topic    string `json:"topic" yaml:"topic" protobuf:"bytes,2,opt,name=topic"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}

type RedisMonitor struct {
	Host     string `json:"host" yaml:"host" protobuf:"bytes,1,opt,name=host"`
	Database string `json:"database" yaml:"database" protobuf:"bytes,2,opt,name=database"`
	Username string `json:"username" yaml:"username" protobuf:"bytes,3,opt,name=username"`
	Password string `json:"password" yaml:"password" protobuf:"bytes,4,opt,name=password"`
}
