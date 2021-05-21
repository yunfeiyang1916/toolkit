package consul

import (
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/yunfeiyang1916/toolkit/logging"
)

// watchKV monitors a key in the KV store for changes.
// The intended use case is to add additional route commands to the routing table.
func watchKV(logger *logging.Logger, client *api.Client, path string, config chan string) {
	var lastIndex uint64
	var lastValue string

	for {
		value, index, err := getKV(client, path, lastIndex)
		if err != nil {
			logger.Warnf("consul: Error fetching config from %s. %v", path, err)
			if strings.Contains(err.Error(), "Unexpected response code: 400") {
				return
			}
			time.Sleep(time.Second)
			continue
		}

		if value != lastValue || index != lastIndex {
			logger.Infof("consul: Manual config changed to #%d (path %s, last value len %d, new value len %d)", index, path, len(lastValue), len(value))
			config <- value
			lastValue, lastIndex = value, index
		}
	}
}

// watchPrefix monitors a prefix in the KV store for changes.
// The intended use case is to add additional route commands to the routing table.
func watchPrefix(logger *logging.Logger, client *api.Client, prefix string, config chan map[string]string) {
	var lastIndex uint64
	var lastValue map[string]string

	for {
		values, index, err := getPrefix(client, prefix, lastIndex)
		if err != nil {
			logger.Warnf("consul: Error fetching prefix config from %s. %v", prefix, err)
			time.Sleep(time.Second)
			continue
		}
		if !reflect.DeepEqual(lastValue, values) || index != lastIndex {
			logger.Infof("consul: Manual prefix config changed to #%d", index)
			config <- values
			lastValue, lastIndex = values, index
		}
	}
}

func getKV(client *api.Client, key string, waitIndex uint64) (string, uint64, error) {
	q := &api.QueryOptions{RequireConsistent: true, WaitIndex: waitIndex}
	kvpair, meta, err := client.KV().Get(key, q)
	if err != nil {
		return "", 0, err
	}
	if kvpair == nil {
		return "", meta.LastIndex, nil
	}
	return strings.TrimSpace(string(kvpair.Value)), meta.LastIndex, nil
}

func getPrefix(client *api.Client, prefix string, waitIndex uint64) (map[string]string, uint64, error) {
	q := &api.QueryOptions{RequireConsistent: true, WaitIndex: waitIndex}
	kvpairs, meta, err := client.KV().List(prefix, q)
	if err != nil {
		return nil, 0, err
	}
	if kvpairs == nil {
		return nil, meta.LastIndex, nil
	}
	kvs := make(map[string]string, len(kvpairs))
	for _, kv := range kvpairs {
		kvs[kv.Key] = strings.TrimSpace(string(kv.Value))
	}
	return kvs, meta.LastIndex, nil
}

func putKV(client *api.Client, key, value string, index uint64) (bool, error) {
	p := &api.KVPair{Key: strings.TrimPrefix(key, "/"), Value: []byte(value), ModifyIndex: index}
	ok, _, err := client.KV().CAS(p, nil)
	if err != nil {
		return false, err
	}
	return ok, nil
}
