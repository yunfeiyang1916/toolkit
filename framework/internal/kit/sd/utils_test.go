package sd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDatacenter(t *testing.T) {
	consulAddr := "test.consul.inkept.cn:8500"
	dc, err := GetDatacenter(consulAddr)
	assert.Equal(t, nil, err)
	assert.Equal(t, "ali-test", dc)
	t.Logf("datacenter:%s", dc)
}
