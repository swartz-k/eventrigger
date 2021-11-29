package monitor

import (
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestParseKafkaMeta(t *testing.T) {
	kafkaOpts := KafkaOptions{
		Servers:           []string{"a", "b", "c"},
		OffsetResetPolicy: earliest,
		Topic:             "topic",
		Version:           sarama.DefaultVersion,
	}
	meta := map[string]string{
		"topic":             kafkaOpts.Topic,
		"servers":           strings.Join(kafkaOpts.Servers, ","),
		"offsetResetPolicy": "earliest",
		"version":           kafkaOpts.Version.String(),
	}

	opt, err := parseKafkaMeta(meta)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, kafkaOpts.Topic, opt.Topic)
	assert.Equal(t, kafkaOpts.Version, opt.Version)
	assert.Equal(t, kafkaOpts.Servers, opt.Servers)
	assert.Equal(t, kafkaOpts.OffsetResetPolicy, opt.OffsetResetPolicy)
}
