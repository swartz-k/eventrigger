package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"strconv"
	"sync"
)

type RedisOptions struct {
	Addr     string
	Username string
	Password string
	DB       int
	Channel  string
}

type RedisRunner struct {
	Opts   *RedisOptions
	StopCh <-chan struct{}
}

func parseRedisMeta(meta map[string]string) (opts *RedisOptions, err error) {
	opts = &RedisOptions{}

	if db, ok := meta["db"]; ok {
		intDB, err := strconv.Atoi(db)
		if err != nil {
			return nil, errors.Wrap(err, "parse meta db to int")
		}
		opts.DB = intDB
	} else {
		opts.DB = 0
	}
	delete(meta, "db")
	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, err
	}

	return opts, nil
}

func NewRedisMonitor(meta map[string]string) (*RedisRunner, error) {
	opts, err := parseRedisMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse redis meta")
	}

	m := &RedisRunner{
		Opts: opts,
	}

	return m, nil
}

func (m *RedisRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {

	rdb := redis.NewClient(&redis.Options{
		Addr:     m.Opts.Addr,
		Username: m.Opts.Username,
		Password: m.Opts.Password,
		DB:       m.Opts.DB,
	})
	pubSub := rdb.Subscribe(ctx, m.Opts.Channel)
	defer pubSub.Close()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)
	go func(pubSub *redis.PubSub) {
		defer waitGroup.Done()
		for msg := range pubSub.Channel() {
			eventChannel <- event.Event{
				Source: msg.Channel,
				Data:   msg.Payload,
			}
		}
	}(pubSub)

	select {
	case <-stopCh:
		fmt.Println("stop kafka")
		return nil
	default:
		waitGroup.Wait()
	}
	return nil
}
