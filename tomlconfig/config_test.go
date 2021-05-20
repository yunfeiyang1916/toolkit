package tomlconfig

import (
	"fmt"
	"testing"
)

type tomlDefaultConfig struct {
	Server struct {
		Name string `toml:"name"`
		Port int    `toml:"port"`
		TCP  struct {
			IdleTimeout       int `toml:"idle_timeout"`
			KeepalinkInterval int `toml:"keepalink_interval"`
		} `toml:"tcp"`
		HTTP struct {
			Location    string `toml:"location"`
			LogResponse string `toml:"log_response"`
		} `toml:"http"`
	} `toml:"server"`
	Trace struct {
		Port int `toml:"port"`
	} `toml:"trace"`
	Log struct {
		Level       string `toml:"level"`
		Businesslog string `toml:"businesslog"`
		Serverlog   string `toml:"serverlog"`
		Accesslog   string `toml:"accesslog"`
		StatLog     string `toml:"statLog"`
		StatMetric  string `toml:"statMetric"`
	} `toml:"log"`
	ServerClient []struct {
		Service     string `toml:"service"`
		Ipport      string `toml:"ipport"`
		Balancetype int    `toml:"balancetype"`
	} `toml:"server_client"`
}

func TestBuildConf(t *testing.T) {

	fmt.Println("start---------buildconfig")

	tomlData := "test.toml"
	confi, err := NewTomlConfig(tomlData)

	if err != nil {

		fmt.Println("build toml config err:", nil)
		return
	}
	// fmt.Println("keys:", confi.KeysLen())

	loglevel, _ := confi.String("loglevel", "default")
	fmt.Println("keys:-", loglevel)

	redis_black, _ := confi.String("redis.black.ip", "default")
	fmt.Println("keys:-", redis_black)

	fmt.Println("end---------buildconfig")

}

func TestInt(t *testing.T) {
	tomlData := "test.toml"
	confi, err := NewTomlConfig(tomlData)

	if err != nil {

		fmt.Println("build toml config err:", nil)
		return
	}

	value, _ := confi.Int64("redis.black.port", 100)
	fmt.Println(value)
	value, _ = confi.Int64("1redis.black.port", 100)

	fmt.Println(value)

	fmt.Println("")
	fmt.Println("end-int")

}

func TestFloat(t *testing.T) {
	tomlData := "test.toml"
	confi, err := NewTomlConfig(tomlData)

	if err != nil {

		fmt.Println("build toml config err:", nil)
		return
	}

	value := confi.Float64("redis.status.port", 100)
	fmt.Println(value)
	value = confi.Float64("1redis.black.port", 100)

	fmt.Println(value)

	fmt.Println("")
	fmt.Println("end-Float64")

}

func TestBool(t *testing.T) {
	tomlData := "test.toml"
	confi, err := NewTomlConfig(tomlData)

	if err != nil {

		fmt.Println("build toml config err:", nil)
		return
	}

	value := confi.Bool("redis.status.print", false)
	fmt.Println("value:", value)
	value = confi.Bool("1redis.status.print", false)
	fmt.Println(value)
}
