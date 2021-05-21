package namespace

import (
	"sync"
	"sync/atomic"
)

type namespaceKey struct{}

var (
	namespaceKeySet sync.Map
	multiNamespace  int32 = 0
)

func AddNamespaceKey(namespace string) {
	if namespace == "" {
		return
	}
	if namespace != LoadtestNamespace {
		atomic.StoreInt32(&multiNamespace, 1)
	}
	namespaceKeySet.Store(namespace, &namespaceKey{})
}

func GetNamespaceKey(namespace string) *namespaceKey {
	if namespace == "" {
		return &namespaceKey{}
	}
	if namespace == LoadtestNamespace {
		return getNamespaceKey(namespace)
	}
	m := atomic.LoadInt32(&multiNamespace)
	if m == 0 {
		return &namespaceKey{}
	}
	return getNamespaceKey(namespace)
}

func getNamespaceKey(namespace string) *namespaceKey {
	val, ok := namespaceKeySet.Load(namespace)
	if ok {
		return val.(*namespaceKey)
	}
	return nil
}
