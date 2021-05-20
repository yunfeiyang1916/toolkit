package tomlconfig

import (
	"strings"

	"github.com/yunfeiyang1916/toolkit/toml"
)

type Config struct {
	meta toml.MetaData
}

var (
	Integer   = "Integer"
	Float     = "Float"
	Datetime  = "Datetime"
	String    = "String"
	Bool      = "Bool"
	Array     = "Array"
	Hash      = "Hash"
	ArrayHash = "ArrayHash"
)

func ParseTomlString(data string, v interface{}) error {
	_, err := toml.Decode(data, v)
	if err != nil {
		return err
	}
	return nil
}

func ParseTomlConfig(filepath string, v interface{}) error {
	_, err := toml.DecodeFile(filepath, v)

	if err != nil {
		return err
	}
	return nil
}

func NewConfig(filepath string, v interface{}) (*Config, error) {

	// fmt.Println("vendor...NewConfig-utils")

	metaData, err := toml.DecodeFile(filepath, v)
	if err != nil {
		return nil, err
	}

	return &Config{
		meta: metaData,
	}, nil
}

func NewTomlConfig(filepath string) (*Config, error) {

	// fmt.Println("vendor...NewTomlConfig-utils")

	var v interface{}
	metaData, err := toml.DecodeFile(filepath, &v)
	if err != nil {
		return nil, err
	}

	return &Config{
		meta: metaData,
	}, nil
}

func (c *Config) String(key string, defaultValue string) (string, bool) {

	keys := strings.Split(key, ".")

	if c.meta.IsDefined(keys...) && c.meta.Type(keys...) == String {
		value := c.meta.FindValue(keys...)
		return value.(string), true
	}
	return defaultValue, false
}

func (c *Config) Int64(key string, defaultValue int64) (int64, bool) {

	keys := strings.Split(key, ".")

	if c.meta.IsDefined(keys...) && c.meta.Type(keys...) == Integer {
		value := c.meta.FindValue(keys...)
		return value.(int64), true
	}

	return defaultValue, false
}

func (c *Config) Float64(key string, defaultValue float64) float64 {

	keys := strings.Split(key, ".")

	if c.meta.IsDefined(keys...) && c.meta.Type(keys...) == Float {
		value := c.meta.FindValue(keys...)
		return value.(float64)
	}

	return defaultValue
}

func (c *Config) Bool(key string, defaultValue bool) bool {

	keys := strings.Split(key, ".")

	if c.meta.IsDefined(keys...) && c.meta.Type(keys...) == Bool {
		value := c.meta.FindValue(keys...)
		return value.(bool)
	}

	return defaultValue
}
