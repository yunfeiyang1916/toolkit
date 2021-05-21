package circuit

import (
	"sync"
	"time"
)

var circuitSetting = make(map[string]*setting) // string -> Setting
var settingsMutex sync.RWMutex

type setting struct {
	SystemLoads struct {
		Open      bool
		Threshold float64
	}

	MaxConcurrent struct {
		Open      bool
		Threshold int64
	}

	QPSLimit struct {
		Open      bool
		Strategy  string
		Threshold int64
	}

	ErrorPercent struct {
		Open       bool
		Threshold  int64
		MinSamples int64
	}

	ConsecutiveError struct {
		Open      bool
		Threshold int64
	}

	AverageRT struct {
		Open bool
		RT   time.Duration
	}
}

// SettingSystemLoads set the system load limiter's configure.
// All SettingXX function can change configure concurrently at runtime.
// Parameter name is the resource name you what to setting.
// Parameter open control whether use system load limiter.
// Parameter threshold is the system load1 value.
func SettingSystemLoads(name string, open bool, threshold float64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.SystemLoads.Open = open
	setting.SystemLoads.Threshold = threshold
}

// SettingMaxConcurrent set the max concurrent limiter's configure.
func SettingMaxConcurrent(name string, open bool, threshold int64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.MaxConcurrent.Open = open
	setting.MaxConcurrent.Threshold = threshold
}

func SettingQPSLimitReject(name string, open bool, reject int64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.QPSLimit.Open = open
	setting.QPSLimit.Strategy = QPSReject
	setting.QPSLimit.Threshold = reject
}

func SettingQPSLimitLeakyBucket(name string, open bool, leaky int64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.QPSLimit.Open = open
	setting.QPSLimit.Strategy = QPSLeakyBucket
	setting.QPSLimit.Threshold = leaky
}

func SettingErrorPercent(name string, open bool, threshold int64, minSamples int64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.ErrorPercent.Open = open
	setting.ErrorPercent.Threshold = threshold
	setting.ErrorPercent.MinSamples = minSamples
}

func SettingConsecutiveError(name string, open bool, threshold int64) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.ConsecutiveError.Open = open
	setting.ConsecutiveError.Threshold = threshold
}

func SettingAverageRT(name string, open bool, rt time.Duration) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	if _, ok := circuitSetting[name]; !ok {
		circuitSetting[name] = &setting{}
	}
	setting := circuitSetting[name]
	setting.AverageRT.Open = open
	setting.AverageRT.RT = rt
}

func getSetting(name string) *setting {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()

	setting, ok := circuitSetting[name]
	if !ok {
		return nil
	}
	t := *setting
	return &t
}
