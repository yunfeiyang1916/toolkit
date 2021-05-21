// Package config is an interface for dynamic configuration.
package config

import (
	"github.com/yunfeiyang1916/toolkit/framework/config/loader"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/sd"
)

// Config represents a config instance
type Config interface {
	reader.Values
	LoadFile(f ...string) error
	LoadPath(p string, isPrefix bool, format string) error
	Sync() error
	Listen(interface{}) loader.Refresher
}

// Default is a default config instance
var Default = New()

// New make a new config
func New(opts ...Option) Config {
	return newDefaultConfig(opts...)
}

// DefaultRemotePath
func DefaultRemotePath(sdname, path string) string {
	remotePath, _ := sd.RegistryKVPath(sdname, path)
	return remotePath
}

// Bytes wrap Default's Bytes func
func Bytes() []byte {
	return Default.Bytes()
}

// String wrap Default's String func
func String() string {
	return Default.String()
}

// Range wrap Default's Range func
func Range(f func(k string, v interface{})) {
	Default.Range(f)
}

// Map wrap Default's Map func
func Map() map[string]interface{} {
	return Default.Map()
}

// Scan wrap Default's Scan func
func Scan(v interface{}) error {
	return Default.Scan(v)
}

// Get wrap Default's Get func
func Get(keys ...string) reader.Value {
	return Default.Get(keys...)
}

// Files wrap Default Load func, it's represents Default load multi config files
func Files(files ...string) error {
	return Default.LoadFile(files...)
}

// Consul wrap Default Load func, it's represents Default load multi consul config path
func Consul(paths ...string) error {
	for _, p := range paths {
		if err := Default.LoadPath(p, false, "toml"); err != nil {
			return err
		}
	}
	return nil
}

// Sync wrap Default's Sync func, use for reloading config
func Sync() error {
	return Default.Sync()
}

// Listen wrap Default's Listen func, use for watching a struct data which maybe change anytime
func Listen(structPtr interface{}) loader.Refresher {
	return Default.Listen(structPtr)
}
