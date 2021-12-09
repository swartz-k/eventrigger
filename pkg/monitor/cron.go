package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
)

type CronOptions struct {
	Cron string
}

type CronMonitor struct {
	Opts       CronOptions
	Cron       *cron.Cron
	Scheduler  cron.Schedule
	CronMapper map[cron.EntryID]chan struct{}
	StopCh     chan struct{}
}

func parseCronMeta(meta map[string]string) (*CronOptions, error) {
	opts := &CronOptions{}
	err := mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "fail parse cron meta %s", meta)
	}

	if opts == nil || opts.Cron == "" {
		return nil, errors.New("cron opts cron is empty")
	}
	return opts, nil
}

func NewCronMonitor(meta map[string]string) (*CronMonitor, error) {
	opts, err := parseCronMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse cron meta")
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	scheduler, err := parser.Parse(opts.Cron)
	if err != nil {
		return nil, err
	}
	c := cron.New(cron.WithSeconds())
	m := &CronMonitor{
		Opts:       *opts,
		Scheduler:  scheduler,
		Cron:       c,
		CronMapper: map[cron.EntryID]chan struct{}{},
		StopCh: make(chan struct{}),
	}

	return m, nil
}

func (m *CronMonitor) Run(ctx context.Context, eventChannel chan event.Event) error {
	fStopCh := make(chan struct{})
	entryId, err := m.Cron.AddFunc(m.Opts.Cron, func() {
		ev := event.Event{
			Source: "",
			Type:   "cron",
			Data:   strconv.Itoa(int(time.Now().UnixNano())),
		}
		eventChannel <- ev
		select {
		case <-fStopCh:
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
	case <- m.StopCh:
		for _, ch := range m.CronMapper {
			ch <- struct{}{}
		}
		m.Cron.Stop()
		fmt.Println("stop cron")
	}
	return nil
}

func (m *CronMonitor) Stop() error {
	m.StopCh <- struct{}{}
	return nil
}