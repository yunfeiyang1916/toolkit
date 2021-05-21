package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	uuid "github.com/satori/go.uuid"
	utils "github.com/yunfeiyang1916/toolkit/utils"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
)

const (
	headersMessageIDKey = "@m_id"
	headersCreateAtKey  = "@m_create"
	headersNamespaceKey = "@m_ns"
)

type nsKeyType struct{}

var nskey = nsKeyType{}

func WithNSKey(ctx context.Context, ns string) context.Context {
	return context.WithValue(ctx, nskey, ns)
}

func NSKey(ctx context.Context) (string, bool) {
	ns, ok := ctx.Value(nskey).(string)
	return ns, ok
}

type SendResponse struct {
	Partition int32
	Offset    int64
	Err       error
}

type sendMeta struct {
	eventTime time.Time
	span      opentracing.Span
	mid       string
	oldMeta   interface{}
}

// Send message to kafka cluster, ctx is http/rpc context
// headers rfs = https://cwiki.apache.org/confluence/display/KAFKA/KIP-82+-+Add+Record+Headers
func (ksc *KafkaSyncClient) Send(ctx context.Context, message *ProducerMessage) (int32, int64, error) {
	span, msg, _ := generateMessageSpan(ctx, message, ksc.headerSupported)
	ext.Component.Set(span, "inkelogic/go-kafkaproducer-sync")
	ext.PeerService.Set(span, ksc.conf.ProducerTo)
	ext.PeerAddress.Set(span, ksc.conf.Broken)

	st := utils.NewServiceStatEntry(P_KAFKA_PRE, ksc.conf.ProducerTo)
	span.LogFields(
		opentracinglog.String("event", "ProduceMessage"),
		opentracinglog.String("mid", message.MessageID))
	partition, offset, err := ksc.producter.SendMessage(msg)
	st.End("Topic."+message.Topic+".SyncSendStatus", finishMessageSpan(span, partition, offset, err))
	return partition, offset, err
}

func (ksc *KafkaClient) Send(ctx context.Context, message *ProducerMessage) (int32, int64, error) {
	span, msg, now := generateMessageSpan(ctx, message, ksc.headerSupported)
	ext.Component.Set(span, "inkelogic/go-kafkaproducer-async")
	ext.PeerService.Set(span, ksc.conf.ProducerTo)
	ext.PeerAddress.Set(span, ksc.conf.Broken)
	st := utils.NewServiceStatEntry(P_KAFKA_PRE, ksc.conf.ProducerTo)
	span.LogFields(
		opentracinglog.String("event", "ProduceMessage"),
		opentracinglog.String("mid", message.MessageID))
	msg.Metadata = &sendMeta{eventTime: now, span: span, mid: message.MessageID, oldMeta: message.Metadata}
	(ksc.producer).Input() <- msg
	st.End("Topic."+message.Topic+".ASyncSendStatus", KafkaSuccess)
	return -1, -1, nil
}

func (kc *KafkaSyncClient) Close() error {
	return kc.producter.Close()
}

func generateMessageSpan(ctx context.Context, message *ProducerMessage, headerSupported bool) (opentracing.Span, *sarama.ProducerMessage, time.Time) {
	span, _ := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("Kafka Producer %s", message.Topic))
	ext.SpanKindProducer.Set(span)
	msg := &sarama.ProducerMessage{}
	now := time.Now()
	if headerSupported {
		carrier := opentracing.TextMapCarrier{}
		span.Tracer().Inject(span.Context(), opentracing.TextMap, carrier)
		if message.MessageID == "" {
			msgID := strings.SplitN(fmt.Sprintf("%s", span.Context()), ":", 2)[0]
			message.MessageID = msgID
		}
		carrier[headersMessageIDKey] = message.MessageID
		carrier[headersCreateAtKey] = fmt.Sprintf("%d", now.UnixNano())

		if ns, _ := ctx.Value(nskey).(string); ns != "" {
			carrier[headersNamespaceKey] = ns
		}

		headers := make([]sarama.RecordHeader, 0, len(carrier))
		for k, v := range carrier {
			headers = append(headers,
				sarama.RecordHeader{
					Key:   []byte(k),
					Value: []byte(v),
				},
			)
		}
		msg.Headers = headers
	}

	msg.Topic = message.Topic
	if message.Partition <= 0 {
		msg.Partition = int32(-1)
	}
	if message.Key == "" {
		// 版本变化
		// u1, _ := uuid.NewV4()
		u1 := uuid.NewV4()
		msg.Key = sarama.ByteEncoder(u1.Bytes())
	} else {
		msg.Key = sarama.StringEncoder(message.Key)
	}
	msg.Value = sarama.ByteEncoder(message.Value)
	if message.Timestamp.IsZero() {
		msg.Timestamp = now
	} else {
		msg.Timestamp = message.Timestamp
	}
	return span, msg, now
}

func finishMessageSpan(span opentracing.Span, partition int32, offset int64, err error) (code int) {
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(
			opentracinglog.String("event", "SendMessage"),
			opentracinglog.Error(err))
		code = KafkaSendError
	} else {
		span.LogFields(
			opentracinglog.String("event", "SendMessage"),
			opentracinglog.Int32("partition", partition),
			opentracinglog.Int64("offset", offset),
		)
		code = KafkaSuccess
	}
	span.Finish()
	return
}
