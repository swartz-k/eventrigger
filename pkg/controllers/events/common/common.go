package common

import "context"

type Event struct {
	Source  string `json:"source" yaml:"source"`
	Type    string `json:"type" yaml:"type"`
	Version string `json:"version" yaml:"version"`
	Data    string `json:"data" yaml:"data"`
}

type Monitor struct {
	ActType   string `json:"act_type" yaml:"act_type"`   // ActType ADD DELETE
	Namespace string `json:"namespace" yaml:"namespace"` // opt, in k8s event needed
	Source    string `json:"source" yaml:"source"`
	Type      string `json:"type" yaml:"type"`
	Version   string `json:"version" yaml:"version"`
}

// Controller for listen and receive events like request, eg: cloud events、kubernetes events
type Controller interface {
	Run(ctx context.Context, monitorChannel chan Monitor) error
}
