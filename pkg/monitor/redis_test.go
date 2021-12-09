package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestParseRedisMeta(t *testing.T) {
	redisOpts := RedisOptions{
		Addr:     "127.0.0.1:6379",
		Username: "default",
		Password: "password",
		DB:       0,
		Channel:  "test",
	}
	meta := map[string]string{
		"addr":     redisOpts.Addr,
		"username": redisOpts.Username,
		"password": redisOpts.Password,
		"db":       strconv.Itoa(redisOpts.DB),
		"channel":  redisOpts.Channel,
	}
	opts, err := parseRedisMeta(meta)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, redisOpts.Addr, opts.Addr)
	assert.Equal(t, redisOpts.Username, opts.Username)
	assert.Equal(t, redisOpts.Password, opts.Password)
	assert.Equal(t, redisOpts.DB, opts.DB)
	assert.Equal(t, redisOpts.Channel, opts.Channel)
}

func TestRedisRunnerRun(t *testing.T) {
	redisOpts := &RedisOptions{
		Addr:     "127.0.0.1:6379",
		Username: "username",
		Password: "password",
		DB:       0,
		Channel:  "test",
	}
	ctx := context.Background()
	stopCh := make(chan struct{})
	eventChannel := make(chan event.Event)
	redisRunner := RedisMonitor{Opts: redisOpts}
	go redisRunner.Run(ctx, eventChannel, stopCh)

	for {
		select {
		case event, ok := <-eventChannel:
			if ok {
				t.Logf("receive event %v, event %s \n", ok, event.Data)
			} else {
				t.Logf("receive event %v \n", ok)
			}
		case <-stopCh:
			return
		}
	}
}
