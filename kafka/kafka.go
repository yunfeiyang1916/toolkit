package kafka

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/samuel/go-zookeeper/zk"
	uuid "github.com/satori/go.uuid"
	log "github.com/yunfeiyang1916/toolkit/logging"
	utils "github.com/yunfeiyang1916/toolkit/utils"
	"go.uber.org/zap"
)

const (
	P_KAFKA_PRE               string = "kfkclient"
	C_KAFKA_PRE               string = "ckfklient"
	KAFKA_INIT                string = "Init"
	KAFKA_GET_PRODUCER_CLIENT string = "PCGet"
	KAFKA_GET_CONSUME_CLIENT  string = "CCGet"
)

var (
	KAFKA_CLIENT_NOT_INIT = errors.New("kafka client not init")
	KAFKA_PARAMS_ERROR    = errors.New("kafka params error")
)

var (
	REQUIRED_ACK_NO_RESPONSE    string = "NoResponse"
	REQUIRED_ACK_WAIT_FOR_LOCAL string = "WaitForLocal"
	REQUIRED_ACK_WAIT_FOR_ALL   string = "WaitForAll"
)

var (
	// Logger is kafka client logger
	Logger         = stdlog.New(ioutil.Discard, "[Sarama] ", stdlog.LstdFlags)
	loggerInitOnce = &sync.Once{}
	noopTracer     = opentracing.NoopTracer{}
)

type KafkaProductConfig struct {
	ProducerTo     string `toml:"producer_to"`
	Broken         string `toml:"kafka_broken"`
	RetryMax       int    `toml:"retrymax"`
	RequiredAcks   string `toml:"RequiredAcks"`
	GetError       bool   `toml:"get_error"`
	GetSuccess     bool   `toml:"get_success"`
	RequestTimeout int    `toml:"request_timeout"`
	Printf         bool
	UseSync        bool
}

type KafkaClient struct {
	producer        sarama.AsyncProducer
	conf            KafkaProductConfig
	perror          chan *ProducerError
	pmessage        chan *ProducerMessage
	headerSupported bool
}

type KafkaSyncClient struct {
	producter       sarama.SyncProducer
	conf            KafkaProductConfig
	headerSupported bool
}

// ProducerMessage is the collection of elements passed to the Producer in order to send a message.
type ProducerMessage struct {
	Topic string // The Kafka topic for this message.
	// The partitioning key for this message. Pre-existing Encoders include
	// StringEncoder and ByteEncoder.
	Key string
	// The actual message to store in Kafka. Pre-existing Encoders include
	// StringEncoder and ByteEncoder.
	Value []byte

	// This field is used to hold arbitrary data you wish to include so it
	// will be available when receiving on the Successes and Errors channels.
	// Sarama completely ignores this field and is only to be used for
	// pass-through data.
	Metadata interface{}

	// Below this point are filled in by the producer as the message is processed

	// Offset is the offset of the message stored on the broker. This is only
	// guaranteed to be defined if the message was successfully delivered and
	// RequiredAcks is not NoResponse.
	Offset int64
	// Partition is the partition that the message was sent to. This is only
	// guaranteed to be defined if the message was successfully delivered.
	Partition int32
	// Timestamp is the timestamp assigned to the message by the broker. This
	// is only guaranteed to be defined if the message was successfully
	// delivered, RequiredAcks is not NoResponse, and the Kafka broker is at
	// least version 0.10.0.
	Timestamp time.Time

	// MessageID
	MessageID string
}

func init() {
	sarama.Logger = Logger
	// https://github.com/Shopify/sarama/issues/959
	sarama.MaxRequestSize = 1000000
}

type logWriter struct {
	l *log.Logger
}

func (l *logWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	if l.l != nil {
		if bytes.Contains(p, []byte("err")) || bytes.Contains(p, []byte("FAILED")) {
			l.l.Error(string(p))
		} else if bytes.Contains(p, []byte("must")) || bytes.Contains(p, []byte("should")) {
			l.l.Warn(string(p))
		} else {
			l.l.Info(string(p))
		}
	}
	return len(p), nil
}

func initLogger() {
	loggerInitOnce.Do(func() {
		if sarama.Logger == Logger {
			sarama.Logger = stdlog.New(&logWriter{l: &log.Logger{SugaredLogger: log.Log(log.GenLoggerName).SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(2)).Sugar()}}, "kafka ", 0)
			zk.DefaultLogger = stdlog.New(&logWriter{l: &log.Logger{SugaredLogger: log.Log(log.GenLoggerName).SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(2)).Sugar()}}, "zookeeper ", 0)
		}
	})
}

// ProducerError is the type of error generated when the producer fails to deliver a message.
// It contains the original ProducerMessage as well as the actual error value.
type ProducerError struct {
	Msg *ProducerMessage
	Err error
}

func makeProducterMsg(spmsg *sarama.ProducerMessage) *ProducerMessage {

	key, _ := spmsg.Key.Encode()
	value, _ := spmsg.Value.Encode()

	return &ProducerMessage{
		Topic:     spmsg.Topic,
		Key:       string(key),
		Value:     value,
		Metadata:  spmsg.Metadata,
		Offset:    spmsg.Offset,
		Partition: spmsg.Partition,
		Timestamp: spmsg.Timestamp,
	}
}

func makeProducterError(sperror *sarama.ProducerError) *ProducerError {

	pm := makeProducterMsg(sperror.Msg)

	return &ProducerError{
		Msg: pm,
		Err: sperror.Err,
	}
}

// getRequiredAcks 如果默认不配置 acks，使用 acks=-1 防止消息丢失
func getRequiredAcks(conf KafkaProductConfig) (sarama.RequiredAcks, error) {
	if len(conf.RequiredAcks) == 0 {
		conf.RequiredAcks = REQUIRED_ACK_WAIT_FOR_ALL
	}
	if conf.RequiredAcks == REQUIRED_ACK_NO_RESPONSE {
		return sarama.NoResponse, nil
	}
	if conf.RequiredAcks == REQUIRED_ACK_WAIT_FOR_ALL {
		return sarama.WaitForAll, nil
	}
	if conf.RequiredAcks == REQUIRED_ACK_WAIT_FOR_LOCAL {
		return sarama.WaitForLocal, nil
	}
	return 0, KAFKA_PARAMS_ERROR
}

func NewSyncProducterClient(conf KafkaProductConfig) (*KafkaSyncClient, error) {
	initLogger()
	config := sarama.NewConfig()
	acks, errf := getRequiredAcks(conf)
	if errf != nil {
		return nil, errf
	}
	config.Net.KeepAlive = 60 * time.Second
	config.Producer.RequiredAcks = acks
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.MaxMessageBytes = int(sarama.MaxRequestSize - 1) // 1M
	if conf.RequestTimeout == 0 {
		config.Producer.Timeout = 5 * time.Second
	} else {
		config.Producer.Timeout = time.Duration(conf.RequestTimeout) * time.Second
	}
	brokerList := strings.Split(conf.Broken, ",")
	if err := adjustProducerVersion(brokerList, config); err != nil {
		return nil, err
	}
	headerSupported := false
	if config.Version.IsAtLeast(sarama.V0_11_0_0) {
		headerSupported = true
	}
	p, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		log.GenLogf("Failed to produce message :%s", err)
		return nil, err
	}

	return &KafkaSyncClient{
		producter:       p,
		conf:            conf,
		headerSupported: headerSupported,
	}, nil
}

func NewKafkaClient(conf KafkaProductConfig) (*KafkaClient, error) {
	initLogger()
	log.GenLog("kafka_util,nomal,producter,new kafka client,broken:", conf.Broken, ",productTo:", conf.ProducerTo, ",retryMax:", conf.RetryMax)
	brokerList := strings.Split(conf.Broken, ",")
	config := sarama.NewConfig()
	acks, errf := getRequiredAcks(conf)
	if errf != nil {
		return nil, errf
	}

	config.Net.KeepAlive = 60 * time.Second
	config.Producer.RequiredAcks = acks
	config.Producer.Retry.Max = conf.RetryMax + 1
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	config.Producer.MaxMessageBytes = int(sarama.MaxRequestSize - 1) // 1M

	if err := adjustProducerVersion(brokerList, config); err != nil {
		return nil, err
	}
	headerSupported := false
	if config.Version.IsAtLeast(sarama.V0_11_0_0) {
		headerSupported = true
	}

	producer, err := sarama.NewAsyncProducer(brokerList, config)
	// producer := &p

	if err != nil {
		errf := fmt.Errorf("init syncProcycer error %s", err.Error())
		log.GenLog("kafka_util,error,producter,init error ,broken:", conf.Broken, ",productTo:", conf.ProducerTo, ",retryMax:", conf.RetryMax, ",err:", err.Error())
		return nil, errf
	}

	kc := &KafkaClient{
		producer:        producer,
		conf:            conf,
		perror:          make(chan *ProducerError),
		pmessage:        make(chan *ProducerMessage),
		headerSupported: headerSupported,
	}

	go func() {

		errChan := producer.Errors()
		successChan := producer.Successes()
		for {

			select {
			case perr, ok := <-errChan:
				if !ok {
					return
				}
				meta := perr.Msg.Metadata.(*sendMeta)
				if meta.oldMeta != nil || meta.mid != "" {
					log.Warnf("[KafkaProducer] send message to %s error %s, brokers(%s), meta(%v), id(%s)", conf.ProducerTo, perr.Error(), conf.Broken, meta.oldMeta, meta.mid)
				} else {
					log.Warnf("[KafkaProducer] send message to %s error %s, brokers(%s)", conf.ProducerTo, perr.Error(), conf.Broken)
				}
				utils.ReportServiceEvent(P_KAFKA_PRE, conf.ProducerTo, "Topic."+perr.Msg.Topic+".AsyncSendStats", meta.eventTime, time.Now(),
					finishMessageSpan(meta.span, -1, -1, perr.Err))
				perr.Msg.Metadata = meta.oldMeta
				if conf.GetError == true {
					kc.perror <- makeProducterError(perr)
				}
			case succ, ok := <-successChan:
				if !ok {
					return
				}
				meta := succ.Metadata.(*sendMeta)
				utils.ReportServiceEvent(P_KAFKA_PRE, conf.ProducerTo, "Topic."+succ.Topic+".AsyncSendStats", meta.eventTime, time.Now(),
					finishMessageSpan(meta.span, succ.Partition, succ.Offset, nil))
				succ.Metadata = meta.oldMeta
				if conf.GetSuccess == true {
					msg := makeProducterMsg(succ)
					kc.pmessage <- msg
				} else {
					if meta.mid != "" || meta.oldMeta != nil {
						log.Infof("send message id %q partition %d, offset %d", meta.mid, succ.Partition, succ.Offset)
					}
				}
			}
		}
	}()
	return kc, nil
}

// Deprecated: sendmsg func should not use anymore.
func (kc *KafkaClient) sendmsg(topic string, key string, msg []byte) error {
	(kc.producer).Input() <- &sarama.ProducerMessage{
		Topic:    topic,
		Value:    sarama.ByteEncoder(msg),
		Key:      sarama.StringEncoder(key),
		Metadata: &sendMeta{eventTime: time.Now(), span: noopTracer.StartSpan(""), mid: "", oldMeta: nil},
	}
	return nil
}

// Deprecated: SendSyncMsg func should not use anymore.
// Use Send func instead
func (ksc *KafkaSyncClient) SendSyncMsg(topic, key string, msg []byte) (int32, int64, error) {

	msgg := &sarama.ProducerMessage{}
	msgg.Topic = topic
	msgg.Partition = int32(-1)
	msgg.Key = sarama.StringEncoder(key)
	msgg.Value = sarama.ByteEncoder(msg)
	if ksc.producter == nil {
		return 0, 0, fmt.Errorf("sync client not init,topic:%v", topic)
	}
	st := utils.NewServiceStatEntry(P_KAFKA_PRE, ksc.conf.ProducerTo)
	partition, offset, err := ksc.producter.SendMessage(msgg)
	code := KafkaSuccess
	if err != nil {
		code = KafkaSendError
	}
	st.End(topic+".SyncSendStatus", code)
	return partition, offset, err
}

// Deprecated: SendKeyMsg func should not use anymore.
// Use Send func instead
func (kc *KafkaClient) SendKeyMsg(topic string, key string, msg []byte) error {
	remoteService := kc.conf.ProducerTo
	stCode := KafkaSuccess
	st := utils.NewServiceStatEntry(P_KAFKA_PRE, remoteService)
	defer st.End("Topic."+topic, stCode)
	if kc.producer == nil {
		log.GenLog("kafka_util,error,producter,send msg ,producer nil,,producter to:", kc.conf.ProducerTo, ",topic:", topic, ",msg:", string(msg))
		stCode = KafkaSendNotInit
		return KAFKA_CLIENT_NOT_INIT
	}
	kc.sendmsg(topic, key, msg)
	return nil

}

// Deprecated: SendMsg func should not use anymore.
// Use Send func instead
func (kc *KafkaClient) SendMsg(topic string, msg []byte) error {
	//u1, _ := uuid.NewV4()
	u1 := uuid.NewV4()
	return kc.SendKeyMsg(topic, u1.String(), msg)
}

func (kc *KafkaClient) Close() error {
	defer func() {
		if err := recover(); err != nil {
			// handler(err)
			log.Errorf("function run panic", err)
		}
	}()
	close(kc.perror)
	close(kc.pmessage)
	return (kc.producer).Close()
}

func (kc *KafkaClient) Errors() <-chan *ProducerError {
	return kc.perror
}

func (kc *KafkaClient) Success() <-chan *ProducerMessage {
	return kc.pmessage
}
