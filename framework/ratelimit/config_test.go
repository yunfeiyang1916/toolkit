package ratelimit

import (
	_ "fmt"
	"testing"
	_ "time"

	"github.com/magiconair/properties/assert"

	_ "github.com/yunfeiyang1916/toolkit/framework/config"
	_ "github.com/yunfeiyang1916/toolkit/framework/log"
)

func TestConfig(t *testing.T) {
	/*
		namespace := config.NewNamespace("/service_config/test")
		c := NewConfig(namespace, log.Stdout())
		limt := c.Limter("limiter_name")
		for {
			if limt.Allow() {
				fmt.Println("Wait", time.Now())
			}
			time.Sleep(time.Second / 1000)
		}
		time.Sleep(time.Second * 1000)
	*/
}

func testNewConfig() *Config {
	configs := []LimiterConfig{
		LimiterConfig{
			Name:   "@server@inf.sms.base@/phone/encode",
			Limits: 10000,
			Open:   true,
		},
		LimiterConfig{
			Name:   "@client@inf.sms.base@/phone/encode",
			Limits: 10000,
			Open:   true,
		},
	}
	return NewConfig(configs)
}

func TestConfig_GetLimiter(t *testing.T) {
	c := testNewConfig()
	sLim := c.GetLimiter(ServerLimiterType, "", "inf.sms.base", "/phone/encode")
	if sLim == nil {
		t.Fail()
	}
	cLim := c.GetLimiter(ClientLimiterType, "", "inf.sms.base", "/phone/encode")
	if cLim == nil {
		t.Fail()
	}
}

func TestConfig_getLimiterName(t *testing.T) {
	c := testNewConfig()
	sLimName := c.getLimiterName(ServerLimiterType, "", "inf.sms.base", "/phone/encode")
	assert.Equal(t, sLimName, "@server@inf.sms.base@/phone/encode")
	cLimName := c.getLimiterName(ClientLimiterType, "", "inf.sms.base", "/phone/encode")
	assert.Equal(t, cLimName, "@client@inf.sms.base@/phone/encode")
}
