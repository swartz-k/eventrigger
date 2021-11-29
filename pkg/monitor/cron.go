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

type CronRunner struct {
	Opts       CronOptions
	Cron       *cron.Cron
	Scheduler  cron.Schedule
	CronMapper map[cron.EntryID]chan struct{}
	StopCh     <-chan struct{}
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

func NewCronMonitor(meta map[string]string) (*CronRunner, error) {
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
	m := &CronRunner{
		Opts:       *opts,
		Scheduler:  scheduler,
		Cron:       c,
		CronMapper: map[cron.EntryID]chan struct{}{},
	}

	return m, nil
}

func (m *CronRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {
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
	case <-stopCh:
		for _, ch := range m.CronMapper {
			ch <- struct{}{}
		}
		m.Cron.Stop()
		fmt.Println("stop cron")
	}
	return nil
}
