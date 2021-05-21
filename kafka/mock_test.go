package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestProducerReturnsExpectationsToChannels(t *testing.T) {
	syncClient, mock, err := NewMockSyncProducerClient()
	if err != nil {
		return
	}
	defer mock.Close()

	mock.ExpectSendMessageAndSucceed()
	mock.ExpectSendMessageAndSucceed()
	mock.ExpectSendMessageAndFail(sarama.ErrOutOfBrokers)

	partition, offset, err := syncClient.SendSyncMsg("test", "hello", []byte(`world`))
	assert.Equal(t, nil, err)
	assert.Equal(t, int32(0), partition)
	assert.Equal(t, int64(1), offset)

	partition, offset, err = syncClient.SendSyncMsg("test", "hello", []byte(`world`))
	assert.Equal(t, nil, err)
	assert.Equal(t, int32(0), partition)
	assert.Equal(t, int64(2), offset)

	partition, offset, err = syncClient.SendSyncMsg("test", "hello", []byte(`world`))
	assert.Equal(t, sarama.ErrOutOfBrokers, err)
	assert.Equal(t, int32(-1), partition)
	assert.Equal(t, int64(-1), offset)
}
func TestNewMockAsyncProducerClient(t *testing.T) {
	asyncClient, mp, err := NewMockAsyncProducerClient()
	if err != nil {
		return
	}
	defer mp.Close()
	mp.ExpectInputAndSucceed()
	mp.ExpectInputAndSucceed()
	mp.ExpectInputAndFail(sarama.ErrOutOfBrokers)

	asyncClient.SendMsg("test-topic", []byte(`hello`))
	asyncClient.SendMsg("test-topic", []byte(`hello`))
	asyncClient.SendMsg("test-topic", []byte(`hello`))

	msg1 := <-asyncClient.Success()
	msg2 := <-asyncClient.Success()
	var err1 *ProducerError
	select {
	case err1 = <-asyncClient.Errors():
	case <-time.After(1 * time.Second):
		t.Fatal("asyncClient ExpectInputAndFail timeout")
	}

	assert.Equal(t, "test-topic", msg1.Topic)
	assert.Equal(t, "test-topic", msg2.Topic)
	assert.Equal(t, "test-topic", err1.Msg.Topic)
	assert.Equal(t, sarama.ErrOutOfBrokers, err1.Err)
}
