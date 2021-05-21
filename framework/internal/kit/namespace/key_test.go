package namespace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddNamespaceKey(t *testing.T) {
	key := GetNamespaceKey(LoadtestNamespace)
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, true, key == nil)

	key = GetNamespaceKey("gmu")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	key = GetNamespaceKey("haokan")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	AddNamespaceKey(LoadtestNamespace)

	key = GetNamespaceKey(LoadtestNamespace)
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	key = GetNamespaceKey("gmu")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	key = GetNamespaceKey("haokan")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	AddNamespaceKey("gmu")

	key = GetNamespaceKey("gmu")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})

	key = GetNamespaceKey("haokan")
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, true, key == nil)

	key = GetNamespaceKey(LoadtestNamespace)
	fmt.Printf("key:%v\n", key)
	assert.Equal(t, key, &namespaceKey{})
}
