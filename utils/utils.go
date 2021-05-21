package utils

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	goMetrics "github.com/rcrowley/go-metrics"
	metrics "github.com/yunfeiyang1916/toolkit/metrics"
)

var (
	statMetric, _ = os.LookupEnv("STAT_METRIC")
	stFileName, _ = os.LookupEnv("STAT_PATH")
)

const (
	statProjectTag = "project"
	statClientTag  = "clienttag"
)

const (
	SUCC_CODE int = 0
)

var (
	succCodeMap map[int]int
)

func AddSuccCode(code int) {
	succCodeMap[code] = 1
}

func AddSBatchuccCode(codeMap map[int]int) {
	for k, v := range codeMap {
		succCodeMap[k] = v
	}
	metrics.AddSuccessCode(codeMap)
}

//SetStat设置stat日志文件的路径和Metric
func SetStat(stFileName, stMetric string) {
	if stMetric != "" {
		statMetric = stMetric
	}
	metrics.SetDefaultRergistryTags(map[string]string{statProjectTag: stMetric})
	metrics.SetStatOutput(stFileName)
}

func init() {
	succCodeMap = make(map[int]int)
	if statMetric == "" {
		binaryName := strings.Split(os.Args[0], "/")
		statMetric = binaryName[len(binaryName)-1]
	}
	statIntervalEnv, _ := os.LookupEnv("STAT_INTERVAL")
	statInterval, _ := strconv.Atoi(statIntervalEnv)
	if statInterval == 0 {
		statInterval = 60
	}
	go metrics.FalconWithTags(goMetrics.DefaultRegistry, time.Duration(statInterval)*time.Second, map[string]string{statProjectTag: statMetric})
	if stFileName != "" {
		metrics.SetStatOutput(stFileName)
	}
}

// GetLocalIP 获取本机IP
func GetLocalIP() ([]string, error) {
	ret := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ret, err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ret = append(ret, ipnet.IP.String())
			}
		}
	}
	return ret, err
}

// stats对象, 这个对象提供了一些方法用来上报数据
type StatEntry struct {
	end           time.Time
	start         time.Time
	event         string
	category      string
	code          int
	remoteservice string
	tag           string
}

func getEventName(event, category string) string {
	return event + "." + category
}

func getServiceEvent(client, event string) string {
	if len(event) != 0 {
		event = client + "." + event
	} else {
		event = client
	}
	return event
}

// End结束statEntry实例.
// category: 需要监控的metric的尾部,最后上报的metric结果:event.category
// code: 当前metric的 错误码,0代表成功,非0失败
func (st *StatEntry) End(category string, code int) {
	eventName := getEventName(st.event, category)
	if len(st.tag) == 0 {
		metrics.Timer(eventName, st.start, metrics.TagCode, code)
		return
	}
	metrics.Timer(eventName, st.start, metrics.TagCode, code, statClientTag, st.tag)
}

//EndStat 结束st的统计
func EndStat(st *StatEntry, category string, code int) {
	st.End(category, code)
}

// NewStatEntry开始一个statEntry实例, 参数为需要监控的metric信息的首部.
func NewStatEntry(event string) *StatEntry {
	return &StatEntry{
		event: event,
		start: time.Now(),
	}
}

// 在rpc-go内部使用的函数
func NewServiceStatEntry(client string, event string) *StatEntry {
	service := event
	st := &StatEntry{
		event:         getServiceEvent(client, event),
		start:         time.Now(),
		remoteservice: service,
		tag:           client,
	}
	return st
}

//ReportEvent 直接上报一条统计信息
func ReportEvent(event, category string, start, end time.Time, code int) {
	metrics.TimerDuration(getEventName(event, category), end.Sub(start), metrics.TagCode, code)
}

func ReportEventGauge(name string, value int, tags ...interface{}) {
	metrics.Gauge(name, value, tags...)
}

//ReportServiceEvent 直接上报一条统计信息
func ReportServiceEvent(client, event, category string, start, end time.Time, code int) {
	metrics.TimerDuration(getEventName(getServiceEvent(client, event), category), end.Sub(start), metrics.TagCode, code, statClientTag, client)
}
