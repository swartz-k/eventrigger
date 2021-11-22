package monitor

import "eventrigger.com/operator/common/event"

type Runner interface {
	Run(chan event.Event) error
}