package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
)

// TestReporter records producer/consumer errors
type TestReporter struct {
	errors []string
}

func NewTestReporter() *TestReporter {
	return &TestReporter{errors: make([]string, 0)}
}

func (tr *TestReporter) Errorf(format string, args ...interface{}) {
	tr.errors = append(tr.errors, fmt.Sprintf(format, args...))
}

// NewMockSyncProducerClient returns standard sync client ,syncMocker and err.
// syncMocker example shows here: http://github.com/Shopify/sarama/mocks
func NewMockSyncProducerClient() (syncClient *KafkaSyncClient, syncMock *mocks.SyncProducer, err error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	t := NewTestReporter()
	syncProducer := mocks.NewSyncProducer(t, config)
	return &KafkaSyncClient{
		producter:       syncProducer,
		conf:            KafkaProductConfig{},
		headerSupported: true,
	}, syncProducer, nil
}

// NewMockAsyncProducerClient returns standard async client ,asyncMocker and err.
// asyncMocker example shows here: http://github.com/Shopify/sarama/mocks
func NewMockAsyncProducerClient() (asyncClient *KafkaClient, asyncMock *mocks.AsyncProducer, err error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	t := NewTestReporter()
	asyncMock = mocks.NewAsyncProducer(t, config)
	asyncClient = &KafkaClient{
		producer:        asyncMock,
		conf:            KafkaProductConfig{},
		headerSupported: true,
		perror:          make(chan *ProducerError),
		pmessage:        make(chan *ProducerMessage),
	}
	go func() {
		errChan := asyncMock.Errors()
		successChan := asyncMock.Successes()
		for {
			select {
			case perr, ok := <-errChan:
				if !ok {
					return
				}
				asyncClient.perror <- makeProducterError(perr)
			case succ, ok := <-successChan:
				if !ok {
					return
				}
				asyncClient.pmessage <- makeProducterMsg(succ)
			}
		}
	}()
	return
}
