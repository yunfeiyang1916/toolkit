package kafka

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	kazoo "github.com/wvanbergen/kazoo-go"
	tls "github.com/yunfeiyang1916/toolkit/go-tls"
	"github.com/yunfeiyang1916/toolkit/kafka/consumergroup"
	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/utils"
)

var (
	duplicateUsedError = "KafkaConsumerClient: Can't Use Messages and GetMessages Method at the same time."
)

type KafkaConsumeConfig struct {
	ConsumeFrom    string `toml:"consume_from"`
	Zookeeperhost  string `toml:"zkpoints"`
	Topic          string `toml:"topic"`
	Group          string `toml:"group"`
	Initoffset     int    `toml:"initoffset"`
	ProcessTimeout int    `toml:"process_timeout"`
	CommitInterval int    `toml:"commit_interval"`
	GetError       bool   `toml:"get_error"`
	TraceEnable    bool   `toml:"trace_enable"`
	ConsumeAll     bool   `toml:"consume_all"`
}

type KafkaConsumeClient struct {
	consumer    *consumergroup.ConsumerGroup
	conf        KafkaConsumeConfig
	err         chan error
	cloceChan   chan bool
	messageChan chan *ConsumerMessage
	mu          sync.Mutex
}

type RecordHeader struct {
	Key   []byte
	Value []byte
}

type ConsumerMessage struct {
	Key, Value     []byte
	Topic          string
	Partition      int32
	Offset         int64
	Timestamp      time.Time // only set if kafka is version 0.10+, inner message timestamp
	BlockTimestamp time.Time // only set if kafka is version 0.10+, outer (compressed) block timestamp
	MessageID      string
	CreateAt       time.Time
	headers        []*RecordHeader // only set if kafka is version 0.11+
	ctx            context.Context
}

func (m *ConsumerMessage) Context() context.Context {
	tls.SetContext(m.ctx)
	return m.ctx
}

type ConsumeCallback interface {
	Process(values []byte)
}

func NewKafkaConsumeClient(conf KafkaConsumeConfig) (*KafkaConsumeClient, error) {
	initLogger()

	config := consumergroup.NewConfig()
	config.Zookeeper.Logger = sarama.Logger
	config.Net.KeepAlive = 5 * time.Second

	config.Offsets.Initial = int64(conf.Initoffset)
	if conf.ProcessTimeout > 0 { // partition上提交offset的最大超时时间,默认60s
		config.Offsets.ProcessingTimeout = time.Duration(conf.ProcessTimeout) * time.Second
	}
	if conf.CommitInterval > 0 { // 设置自动提交offset间隔,默认10s
		config.Offsets.CommitInterval = time.Duration(conf.CommitInterval) * time.Second
	}
	config.Consumer.Return.Errors = true
	config.Producer.MaxMessageBytes = int(sarama.MaxRequestSize - 1) // 1M

	var zookeeperNodes []string
	zookeeperNodes, config.Zookeeper.Chroot = kazoo.ParseConnectionString(conf.Zookeeperhost)
	err := adjustConsumerVersion(conf.Zookeeperhost, config.Config)
	if err != nil {
		return nil, err
	}
	kafkaTopics := strings.Split(conf.Topic, ",")
	if conf.ConsumeAll {
		hostname, err := os.Hostname()
		if err != nil {
			panic("get hostname error")
		}
		conf.Group = conf.Group + "_" + hostname
	}
	consumer, consumerErr := consumergroup.JoinConsumerGroup(conf.Group, kafkaTopics, zookeeperNodes, config)
	if consumerErr != nil {
		return nil, consumerErr
	}

	kcc := &KafkaConsumeClient{
		consumer:    consumer,
		conf:        conf,
		err:         make(chan error),
		messageChan: nil,
	}

	go func() {
		for err := range consumer.Errors() {
			st := utils.NewServiceStatEntry(C_KAFKA_PRE, conf.ConsumeFrom)
			log.GenLog("kafka_util,error,consume,group:", conf.Group, ",from:", conf.ConsumeFrom, ",topic:", conf.Topic, ",err:", err.Error())
			st.End("consume_inner", KafkaConsumeError)

			if conf.GetError == true {
				kcc.err <- err
			}
		}
	}()
	return kcc, nil
}

func (kcc *KafkaConsumeClient) Close() error {
	kcc.mu.Lock()
	defer kcc.mu.Unlock()
	if kcc.cloceChan != nil {
		close(kcc.cloceChan)
		kcc.cloceChan = nil
	}
	return kcc.consumer.Close()
}

func (kcc *KafkaConsumeClient) Errors() <-chan error {
	return kcc.err
}

func (kcc *KafkaConsumeClient) Messages(closeChan chan bool, maxQueueSize int) chan []byte {
	ch := make(chan []byte, maxQueueSize)
	offsets := make(map[string]map[int32]int64)
	kcc.mu.Lock()
	if kcc.messageChan != nil {
		log.CrashLog(duplicateUsedError)
		panic(duplicateUsedError)
	}
	if kcc.cloceChan == nil {
		if closeChan == nil {
			closeChan = make(chan bool)
		}
		kcc.cloceChan = closeChan
		go func() {
			for {
				select {
				case <-closeChan:
					close(ch)
					return
				case message := <-kcc.consumer.Messages():
					if offsets[message.Topic] == nil {
						offsets[message.Topic] = make(map[int32]int64)
					}

					if offsets[message.Topic][message.Partition] != 0 && offsets[message.Topic][message.Partition] != message.Offset-1 {

					}
					ch <- message.Value
					st := utils.NewServiceStatEntry(C_KAFKA_PRE, kcc.conf.ConsumeFrom)
					kcc.consumer.CommitUpto(message)
					st.End("recver_inner", KafkaSuccess)
					offsets[message.Topic][message.Partition] = message.Offset
				}
			}
		}()
	}
	kcc.mu.Unlock()
	return ch
}

func convertConsumerMessage(ctx context.Context, message *sarama.ConsumerMessage, carrier opentracing.TextMapCarrier) *ConsumerMessage {
	headers := make([]*RecordHeader, len(message.Headers))
	for i, h := range message.Headers {
		headers[i] = &RecordHeader{Key: h.Key, Value: h.Value}
	}
	createAtStr := carrier[headersCreateAtKey]
	createAt, _ := strconv.ParseInt(createAtStr, 10, 64)
	m := &ConsumerMessage{
		Key:            message.Key,
		Value:          message.Value,
		Topic:          message.Topic,
		Partition:      message.Partition,
		Offset:         message.Offset,
		Timestamp:      message.Timestamp,
		BlockTimestamp: message.BlockTimestamp,
		MessageID:      carrier[headersMessageIDKey],
		headers:        headers,
		ctx:            ctx,
	}
	if createAt != 0 {
		m.CreateAt = time.Unix(0, createAt)
	}
	return m
}

func deConvertConsumerMessage(message *ConsumerMessage) *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{
		Topic:     message.Topic,
		Partition: message.Partition,
		Offset:    message.Offset,
	}
}

func (kcc *KafkaConsumeClient) GetMessages() <-chan *ConsumerMessage {
	kcc.mu.Lock()
	if kcc.messageChan == nil {
		if kcc.cloceChan != nil {
			log.CrashLog(duplicateUsedError)
			panic(duplicateUsedError)
		}
		kcc.messageChan = make(chan *ConsumerMessage)
		go func() {
			for {
				st := utils.NewStatEntry(C_KAFKA_PRE)
				message, ok := <-kcc.consumer.Messages()
				if !ok {
					st.End("ConsumeClosed", KafkaSuccess)
					close(kcc.messageChan)
					return
				}
				st.End("Consume."+message.Topic, KafkaSuccess)
				carrier := opentracing.TextMapCarrier{}
				ctx := context.Background()
				for _, h := range message.Headers {
					if k := string(h.Key); k == headersNamespaceKey {
						ctx = WithNSKey(ctx, string(h.Value))
						continue
					}
					carrier[string(h.Key)] = string(h.Value)
				}

				tracer := opentracing.GlobalTracer()
				parent, _ := tracer.Extract(opentracing.TextMap, carrier)
				var span opentracing.Span
				if parent != nil {
					span = tracer.StartSpan(fmt.Sprintf("Kafka Consumer %s", message.Topic), opentracing.ChildOf(parent))
				} else {
					span = tracer.StartSpan(fmt.Sprintf("Kafka Consumer %s", message.Topic))
				}
				ext.SpanKindConsumer.Set(span)
				ext.Component.Set(span, "inkelogic/go-kafka-consumer")
				ext.PeerService.Set(span, kcc.conf.ConsumeFrom)
				ext.PeerAddress.Set(span, kcc.conf.Zookeeperhost)
				span.LogFields(
					opentracinglog.String("event", "RecvMessage"),
					opentracinglog.String("mid", carrier[headersMessageIDKey]),
					opentracinglog.Int32("partition", message.Partition),
					opentracinglog.Int64("offset", message.Offset),
				)
				ctx = opentracing.ContextWithSpan(ctx, span)

				kcc.messageChan <- convertConsumerMessage(ctx, message, carrier)
			}
		}()
	}
	kcc.mu.Unlock()
	return kcc.messageChan
}

func (kcc *KafkaConsumeClient) CommitUpto(message *ConsumerMessage) {
	span := opentracing.SpanFromContext(message.ctx)
	if span != nil {
		tls.Flush()
		span.LogFields(
			opentracinglog.String("mid", message.MessageID),
			opentracinglog.String("event", "MessageCommit"),
			opentracinglog.Int32("partition", message.Partition),
			opentracinglog.Int64("offset", message.Offset),
		)
		span.Finish()
	}
	if !message.CreateAt.IsZero() {
		utils.ReportEvent(C_KAFKA_PRE, fmt.Sprintf("%s.commit_lag", message.Topic), message.CreateAt, time.Now(), 0)
	}
	st := utils.NewStatEntry(C_KAFKA_PRE)
	kcc.consumer.CommitUpto(deConvertConsumerMessage(message))
	st.End("Commit."+message.Topic, 0)
}
