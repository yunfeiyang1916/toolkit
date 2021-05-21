package kafka

import (
	"errors"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	kazoo "github.com/wvanbergen/kazoo-go"
	log "github.com/yunfeiyang1916/toolkit/logging"
)

const (
	versionRequestReadTimeout  = 10 * time.Second
	versionRequestWriteTimeout = 3 * time.Second

	producerHeaderSupportedVersion = 3
	fetchHeaderSupportedVersion    = 4

	produceRequestKey = 0
	fetchRequestKey   = 1
)

const (
	localBigDataKafkaBrokerPrefix = "localkafka"
)

var errConncetedTimeout = errors.New("adjust broker version, connect broker timedout")

// checkKafkaHeadersSupported https://www.confluent.io/blog/upgrading-apache-kafka-clients-just-got-easier/
// https://cwiki.apache.org/confluence/display/KAFKA/KIP-35+-+Retrieving+protocol+version#KIP-35-Retrievingprotocolversion-Aclientdeveloperwantstoaddsupportforanewfeature
// API version request, this requires a >= 0.10.0.0 broker and will cause a disconnect on brokers 0.8.x . NOTE: Due to a bug in broker version 0.9.0.0 & 0.9.0.1 the broker will not close the connection when receiving the API version request, instead the request will time out
func checkKafkaHeadersSupported(addresses []string) (bool, error) {
	for _, b := range addresses {
		if strings.Contains(b, localBigDataKafkaBrokerPrefix) {
			return false, nil
		}
	}
	conf := sarama.NewConfig()
	conf.Net.DialTimeout = versionRequestWriteTimeout
	conf.Net.ReadTimeout = versionRequestReadTimeout
	conf.Net.WriteTimeout = versionRequestWriteTimeout
	conf.Version = sarama.V0_11_0_0
	conf.Metadata.Full = true
	conf.Producer.MaxMessageBytes = int(sarama.MaxRequestSize - 1) // 1M
	client, err := sarama.NewClient(addresses, conf)
	if err != nil {
		return false, err
	}
	defer client.Close()
	for _, br := range client.Brokers() {
		var err error
		supported := 0
		_ = br.Open(conf)
		resp, err := br.ApiVersions(new(sarama.ApiVersionsRequest))
		if err != nil {
			continue
		}
		for _, version := range resp.ApiVersions {
			if version.ApiKey == fetchRequestKey && version.MaxVersion >= fetchHeaderSupportedVersion {
				supported |= 1
				continue
			}
			if version.ApiKey == produceRequestKey && version.MaxVersion >= producerHeaderSupportedVersion {
				supported |= 1 << 1
				continue
			}
		}
		return supported == 3, nil
	}
	return false, nil
}

func getKafkaBrokerListFromZK(zkHosts string) ([]string, error) {
	var zookNodes []string
	zooConfig := kazoo.NewConfig()
	zookNodes, zooConfig.Chroot = kazoo.ParseConnectionString(zkHosts)
	var kz *kazoo.Kazoo
	var err error
	if kz, err = kazoo.NewKazoo(zookNodes, zooConfig); err != nil {
		return nil, err
	}
	brokers, err := kz.BrokerList()
	kz.Close()
	if err != nil {
		return nil, err
	}
	return brokers, nil
}

func adjustProducerVersion(brokers []string, conf *sarama.Config) error {
	supported, err := checkKafkaHeadersSupported(brokers)
	if err != nil {
		log.GenLogf("adjustProducerVersion brokers:%v, error %s", brokers, err)
		return err
	}
	log.GenLogf("adjustProducerVersion brokers:%v, supportedv0_11:%t", brokers, supported)
	log.Infof("adjustProducerVersion brokers:%v, supportedv0_11:%t", brokers, supported)
	if supported {
		conf.Version = sarama.V0_11_0_0
	}
	return nil
}

func adjustConsumerVersion(zkHosts string, conf *sarama.Config) error {
	for _, b := range strings.Split(zkHosts, ",") {
		if strings.Contains(b, localBigDataKafkaBrokerPrefix) {
			return nil
		}
	}
	brokers, err := getKafkaBrokerListFromZK(zkHosts)
	if err != nil {
		log.GenLogf("adjustConsumerVersion get kafka brokers error (zkhosts:%s), error %s", zkHosts, err)
		return err
	}
	supported, err := checkKafkaHeadersSupported(brokers)
	if err != nil {
		return err
	}
	log.GenLogf("adjustConsumerVersion zkhosts:%s, supportedv0_11:%t", zkHosts, supported)
	log.Infof("adjustConsumerVersion zkhosts:%s, supportedv0_11:%t", zkHosts, supported)
	if supported {
		conf.Version = sarama.V0_11_0_0
	}
	return nil
}
