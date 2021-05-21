package kafka

const (
	KafkaSuccess        int = 0   //kafka 0 成功
	KafkaSendInnerError int = 200 //kafka 100 内部错误
	KafkaSendNotInit    int = 201 //kafka 101 未init
	KafkaSendError      int = 202

	KafkaConsumeError      int = 203
	KafkaConsumeInitError  int = 204
	KafkaProducerInitError int = 205

	KafkaGetConsumeClientError  int = 206
	KafkaGetProducerClientError int = 207
)
