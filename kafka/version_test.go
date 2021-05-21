package kafka

import (
	"log"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestCheckKafakHeadersSupported(t *testing.T) {
	// 0.11 cluster
	hosts := []string{"ali-a-inf-kafka-test11.bj:9092", "ali-c-inf-kafka-test12.bj:9092", "ali-a-inf-kafka-test13.bj:9092"}
	supported, err := checkKafkaHeadersSupported(hosts)
	assert.Equal(t, supported, true)
	assert.Equal(t, err, nil)

	// 0.10 cluster
	hosts = []string{"localkafka01.dsj.inke.srv:2181"}
	supported, err = checkKafkaHeadersSupported(hosts)
	log.Printf("checkKafkaHeadersSupported supported=%t, error=%v", supported, err)
	assert.Equal(t, supported, false)
	assert.Equal(t, err, nil)
}

func TestAdjustProducerVersion(t *testing.T) {
	hosts := []string{"ali-a-inf-kafka-test11.bj:9092", "ali-c-inf-kafka-test12.bj:9092", "ali-a-inf-kafka-test13.bj:9092"}
	conf := sarama.NewConfig()
	err := adjustProducerVersion(hosts, conf)
	assert.Equal(t, err, nil)
	assert.Equal(t, conf.Version, sarama.V0_11_0_0)

	hosts = []string{"localkafka01.dsj.inke.srv:9092"}
	conf = sarama.NewConfig()
	err = adjustProducerVersion(hosts, conf)
	assert.Equal(t, err, nil)
	assert.Equal(t, conf.Version, sarama.V0_8_2_0)
}

func TestAdjustConsumerVersion(t *testing.T) {
	conf := sarama.NewConfig()

	err := adjustConsumerVersion("ali-a-inf-kafka-test11.bj:2181,ali-c-inf-kafka-test12.bj:2181,ali-a-inf-kafka-test13.bj:2181,ali-c-inf-kafka-test14.bj:2181,ali-e-inf-kafka-test15.bj:2181/config/inke/inf/mq/kafka-test", conf)
	assert.Equal(t, err, nil)
	assert.Equal(t, conf.Version, sarama.V0_11_0_0)

	conf = sarama.NewConfig()
	err = adjustConsumerVersion("localkafka01.dsj.inke.srv:2181", conf)
	assert.Equal(t, err, nil)
	assert.Equal(t, conf.Version, sarama.V0_8_2_0)
}
