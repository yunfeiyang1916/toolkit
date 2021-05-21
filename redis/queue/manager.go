// 这个package已经过时了， 新代码请不要使用这个package里面得任何api和type
package queue

type QueueManager struct {
	queues map[string]*RedisQueue
}

func NewQueueManager(qc map[string]RedisQueueConfig) *QueueManager {
	qm := QueueManager{queues: map[string]*RedisQueue{}}
	for name, config := range qc {
		qm.queues[name] = NewRedisQueue(config)
	}
	return &qm
}

func (qm *QueueManager) Add(name string, q *RedisQueue) {
	qm.queues[name] = q
}

func (qm *QueueManager) Get(name string) *RedisQueue {
	return qm.queues[name]
}
