package breaker

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func testNewConfig() *Config {
	configs := []BreakerConfig{
		{
			Name:                      "@client@inf.secret.base@*",
			ErrorPercentThreshold:     50,
			ConsecutiveErrorThreshold: 50,
			MinSamples:                100,
			Break:                     false,
		},
		BreakerConfig{
			Name:                      "@server@*",
			ErrorPercentThreshold:     50,
			ConsecutiveErrorThreshold: 50,
			MinSamples:                100,
			Break:                     false,
		},
	}
	return NewConfig(configs)
}

func TestConfig_GetBreaker(t *testing.T) {
	c := testNewConfig()
	sBrk := c.GetBreaker(ServerBreakerType, "", "", "/phone/encode")
	if sBrk == nil {
		t.Fail()
	}

	cBrk := c.GetBreaker(ClientBreakerType, "", "inf.secret.base", "/phone/encode")
	if cBrk == nil {
		t.Fail()
	}
}

func TestConfig_getBreakerName(t *testing.T) {
	c := testNewConfig()
	sBrkName := c.getBreakerName(ServerBreakerType, "", "", "/phone/encode")
	assert.Equal(t, sBrkName, "@server@/phone/encode")

	cBrkName := c.getBreakerName(ClientBreakerType, "", "inf.secret.base", "/phone/encode")
	assert.Equal(t, cBrkName, "@client@inf.secret.base@/phone/encode")
}
