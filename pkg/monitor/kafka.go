package monitor

import (
	"context"
	"eventrigger.com/operator/common/event"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"strings"
	"sync"
)

type offsetResetPolicy string

const (
	latest   offsetResetPolicy = "latest"
	earliest offsetResetPolicy = "earliest"
)

type kafkaSaslType string

// supported SASL types
const (
	KafkaSASLTypeNone        kafkaSaslType = "none"
	KafkaSASLTypePlaintext   kafkaSaslType = "plaintext"
	KafkaSASLTypeSCRAMSHA256 kafkaSaslType = "scram_sha256"
	KafkaSASLTypeSCRAMSHA512 kafkaSaslType = "scram_sha512"
)

const (
	lagThresholdMetricName   = "lagThreshold"
	kafkaMetricType          = "External"
	defaultKafkaLagThreshold = 10
	defaultOffsetResetPolicy = latest
	invalidOffset            = -1
)

type KafkaOptions struct {
	Servers            []string
	ConsumerGroup      string
	Group              string
	Topic              string
	LagThreshold       string
	OffsetResetPolicy  offsetResetPolicy
	AllowIdleConsumers bool
	Version            sarama.KafkaVersion

	// SASL
	SaslType kafkaSaslType
	Username string
	Password string
}

type KafkaRunner struct {
	Opts       *KafkaOptions
	Config     *sarama.Config
	CronMapper map[cron.EntryID]chan struct{}
	StopCh     <-chan struct{}
}

func parseKafkaMeta(meta map[string]string) (opts *KafkaOptions, err error) {
	opts = &KafkaOptions{}

	if servers, ok := meta["servers"]; ok {
		opts.Servers = strings.Split(servers, ",")
		delete(meta, "servers")
	}

	if offsetPolicy, ok := meta["offsetResetPolicy"]; ok {
		switch offsetPolicy {
		case string(earliest):
			opts.OffsetResetPolicy = earliest
		case string(latest):
			opts.OffsetResetPolicy = latest
		default:
			return nil, errors.New(fmt.Sprintf("not supported offsetResetPolicy %s", offsetPolicy))
		}
		delete(meta, "offsetResetPolicy")
	}

	if version, ok := meta["version"]; ok {
		opts.Version, err = sarama.ParseKafkaVersion(version)
		if err != nil {
			return nil, errors.Wrapf(err, "not valid version %s", version)
		}
		delete(meta, "version")
	}

	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, err
	}

	return opts, nil
}

func getKafkaConfig(opts *KafkaOptions) (config *sarama.Config, err error) {
	config = sarama.NewConfig()
	config.Version = opts.Version

	if opts.SaslType != KafkaSASLTypeNone {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = opts.Username
		config.Net.SASL.Password = opts.Password
	}

	if opts.SaslType == KafkaSASLTypePlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	//
	//if opts.saslType == KafkaSASLTypeSCRAMSHA256 {
	//	config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
	//	config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	//}
	//
	//if opts.saslType == KafkaSASLTypeSCRAMSHA512 {
	//	config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
	//	config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	//}
	//
	//client, err := sarama.NewClient(opts.Servers, config)
	//if err != nil {
	//	return nil, nil, fmt.Errorf("error creating kafka client: %s", err)
	//}
	//
	//admin, err := sarama.NewClusterAdminFromClient(client)
	//if err != nil {
	//	if !client.Closed() {
	//		client.Close()
	//	}
	//	return nil, nil, fmt.Errorf("error creating kafka admin: %s", err)
	//}

	return config, nil
}

func NewKafkaMonitor(meta map[string]string) (*KafkaRunner, error) {
	opts, err := parseKafkaMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse kafka meta")
	}
	cfg, err := getKafkaConfig(opts)
	if err != nil {
		return nil, err
	}
	m := &KafkaRunner{
		Opts:   opts,
		Config: cfg,
	}

	return m, nil
}

func (m *KafkaRunner) getTopicOffsets(client sarama.Client, partitions []int32) (map[int32]int64, error) {
	version := int16(0)
	if m.Config.Version.IsAtLeast(sarama.V0_10_1_0) {
		version = 1
	}

	// Step 1: build one OffsetRequest instance per broker.
	requests := make(map[*sarama.Broker]*sarama.OffsetRequest)

	for _, partitionID := range partitions {
		broker, err := client.Leader(m.Opts.Topic, partitionID)
		if err != nil {
			return nil, err
		}

		request, ok := requests[broker]
		if !ok {
			request = &sarama.OffsetRequest{Version: version}
			requests[broker] = request
		}

		request.AddBlock(m.Opts.Topic, partitionID, sarama.OffsetNewest, 1)
	}

	offsets := make(map[int32]int64)

	// Step 2: send requests, one per broker, and collect offsets
	for broker, request := range requests {
		response, err := broker.GetAvailableOffsets(request)

		if err != nil {
			return nil, err
		}

		for _, blocks := range response.Blocks {
			for partitionID, block := range blocks {
				if block.Err != sarama.ErrNoError {
					return nil, block.Err
				}

				offsets[partitionID] = block.Offset
			}
		}
	}

	return offsets, nil
}

func (m *KafkaRunner) Run(ctx context.Context, eventChannel chan event.Event, stopCh <-chan struct{}) error {
	consumer, err := sarama.NewConsumer(m.Opts.Servers, m.Config)
	if err != nil {
		return errors.Wrap(err, "new consumer")
	}
	partitions, err := consumer.Partitions(m.Opts.Topic)
	if err != nil {
		return errors.Wrap(err, "get partition of topic")
	}
	client, err := sarama.NewClient(m.Opts.Servers, m.Config)
	if err != nil {
		return fmt.Errorf("error creating kafka client: %s", err)
	}
	partOffsetMap, err := m.getTopicOffsets(client, partitions)
	if nil != err {
		panic(err)
	}
	waitGroup := sync.WaitGroup{}
	for _, partition := range partitions {
		offsetId, ok := partOffsetMap[partition]
		if !ok {
			zap.L().Warn("")
			break
		}
		pc, err := consumer.ConsumePartition(m.Opts.Topic, partition, offsetId)
		if err != nil {
			zap.L().Info(fmt.Sprintf("failed consume %s-%d-%d", m.Opts.Topic, partition, offsetId))
			break
		}
		waitGroup.Add(1)
		go func(pc sarama.PartitionConsumer) error {
			defer waitGroup.Done()
			for {
				select {
				case message := <-pc.Messages():
					eventChannel <- event.Event{
						Source: message.Topic,
					}
				case <-stopCh:
					return nil
				}
			}
		}(pc)
	}

	select {
	case <-stopCh:
		fmt.Println("stop kafka")
		return nil
	default:
		waitGroup.Wait()
	}
	return nil
}
