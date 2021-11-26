package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
)

type CronOptions struct {
	Cron string
}

type CronRunner struct {
	Opts         CronOptions
	Cron 		*cron.Cron
	Scheduler  cron.Schedule
	CronMapper map[cron.EntryID]chan struct{}
	StopCh <- chan struct{}
}

func NewCronMonitor(opts *CronOptions) (*CronRunner, error) {
	if opts == nil || opts.Cron == "" {
		return nil, errors.New("NewMQTTRunner failed uri username or password is empty")
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	//parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sche, err := parser.Parse(opts.Cron)
	if err != nil {
		return nil, err
	}
	c := cron.New(cron.WithSeconds())
	m := &CronRunner{
		Opts: *opts,
		Scheduler: sche,
		Cron: c,
		CronMapper: map[cron.EntryID]chan struct{}{},
	}

	return m, nil
}

func (m *CronRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <- chan struct{}) error {
	fStopCh := make(chan struct{})
	entryId, err := m.Cron.AddFunc(m.Opts.Cron, func() {
		ev := event.Event{
			Source: "",
			Type: "cron",
			Data: strconv.Itoa(int(time.Now().UnixNano())),
		}
		eventChannel <- ev
		select {
		case <- fStopCh:
			return
		default:
			break
		}
	})
	if err != nil {
		return err
	}
	m.CronMapper[entryId] = fStopCh

	go m.Cron.Run()
	select {
		case <- stopCh:
			for _, ch := range m.CronMapper {
				ch <- struct{}{}
			}
			m.Cron.Stop()
			fmt.Println("stop cron")
	}
	return nil
}