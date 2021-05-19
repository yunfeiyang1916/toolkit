package metrics

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/rcrowley/go-metrics"
	"github.com/yunfeiyang1916/toolkit/rolling"
)

var (
	defaultRegistryInit  int32
	defaultReporterMutex sync.Mutex
	defaultReporter      *reporter
	successCodeMapMutex  sync.Mutex
	successCodeMap       *sync.Map
)

var (
	roll       *rolling.RollingFile
	dataLogger *log.Logger
)

func init() {
	dataLogger = log.New(os.Stderr, "", 0)
	defaultRegistryInit = 0
	successCodeMap = new(sync.Map)
}

func SetStatOutput(name string) {
	if roll != nil {
		roll.Close()
	}
	rf, err := rolling.NewRollingFile(name, rolling.DailyRolling)
	if err != nil {
		return
	}
	roll = rf
	dataLogger = log.New(rf, "", 0)
}

var (
	json             = jsoniter.ConfigCompatibleWithStandardLibrary
	runPid           = os.Getpid()
	localEndPoint, _ = os.Hostname()
)

func AddSuccessCode(cm map[int]int) {
	successCodeMapMutex.Lock()
	defer successCodeMapMutex.Unlock()
	for k := range cm {
		if k == 0 {
			continue
		}
		successCodeMap.Store(k, struct{}{})
	}
}

func ReloadSuccessCode(cm map[int]int) {
	if len(cm) == 0 {
		return
	}
	successCodeMapMutex.Lock()
	defer successCodeMapMutex.Unlock()
	successCodeMap = new(sync.Map)
	for k := range cm {
		if k == 0 {
			continue
		}
		successCodeMap.Store(k, struct{}{})
	}
}

type reporter struct {
	reg      metrics.Registry
	interval time.Duration
	url      string
	tags     unsafe.Pointer
	client   *http.Client
}

var (
	eventCountMap = make(map[string]int64) // 存储上一次counter值
	meterCountMap = make(map[string]int64)
)

const (
	eventTag = "event"
)

const (
	TagCode    = "code"
	TagComment = "comment"
)

const (
	eventCodeCount   = "event.code.count"
	eventCodeAvgtime = "event.code.avgtime"
	eventCodeRatio   = "event.code.rate"
	eventCode0Ratio  = "event.code0.rate" // code0真实成功率

	eventP50 = "event.pt50"
	//	eventP75   = "event.pt75"
	//	eventP95   = "event.pt95"
	eventP99 = "event.pt99"
	//	eventP999  = "event.pt999"
	//	eventP9999 = "event.pt9999"
	//	eventMin   = "event.min"
	//	eventMax   = "event.max"
	//	eventSum   = "event.sum"
	eventAvg = "event.avg"

	eventTimeP50 = "event.time.pt50"
	//	eventTimeP75  = "event.time.pt75"
	//	eventTimeP95  = "event.time.pt95"
	eventTimeP99 = "event.time.pt99"
	//	eventTimeP999 = "event.time.pt999"
	eventTimeMin = "event.time.min"
	eventTimeMax = "event.time.max"
	// eventTimeSum  = "event.time.sum"

	eventTotal = "event.total"
	eventCount = "event.count"
	eventGauge = "event.gauge"
	eventRate  = "event.rate"
	eventRate1 = "event.rate1"
)

type FieldMetadata struct {
	Name    string            `json:"n"`
	Tags    map[string]string `json:"t"`
	Comment map[int]string
}

// ByteSlice2String bytes to string, copy from string.Builder
func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func (f *FieldMetadata) String() string {
	return f.Name + "|" + mapToString(f.Tags, "")
}

func getFieldMetaDataFromString(field string) *FieldMetadata {
	var dat FieldMetadata
	data := strings.SplitN(field, "|", 2)
	dat.Name = data[0]
	if len(data) > 1 {
		tags := strings.Split(data[1], ",")
		var kv []string
		if len(tags) > 0 {
			dat.Tags = make(map[string]string, len(tags))
			for _, t := range tags {
				kv = strings.SplitN(t, "=", 2)
				if len(kv) > 1 {
					dat.Tags[kv[0]] = kv[1]
				}
			}
		}
	}
	dat.Comment = commentGet(data[0])
	return &dat
}

// Falcon starts a Falcon reporter which will post the metrics from the given registry at each d interval.
func Falcon(r metrics.Registry, d time.Duration) {
	FalconWithTags(r, d, nil)
}

// FalconWithTags starts a Falcon reporter which will post the metrics from the given registry at each d interval with the specified tags
func FalconWithTags(r metrics.Registry, d time.Duration, tags map[string]string) {
	copyTags := make(map[string]string, len(tags))
	for k, v := range tags {
		copyTags[k] = v
	}
	if r == metrics.DefaultRegistry {
		if !atomic.CompareAndSwapInt32(&defaultRegistryInit, 0, 1) {
			atomic.StorePointer(&defaultReporter.tags, unsafe.Pointer(&copyTags))
			return
		}
	}

	falconAddr := "http://127.0.0.1:1988/v1/push"
	if v, ok := os.LookupEnv("FALCON_AGENT_ADDR"); ok {
		falconAddr = v
	}
	rep := &reporter{
		reg:      r,
		interval: d,
		url:      falconAddr,
		tags:     unsafe.Pointer(&copyTags),
	}
	defaultReporterMutex.Lock()
	defaultReporter = rep
	defaultReporterMutex.Unlock()
	if err := defaultReporter.makeClient(); err != nil {
		log.Printf("unable to make falcon client. err=%v", err)
		return
	}
	metrics.RegisterRuntimeMemStats(r)
	metrics.RegisterDebugGCStats(r)
	rep.run()
}

func SetDefaultRergistryTags(tags map[string]string) {
	defaultReporterMutex.Lock()
	rep := defaultReporter
	defaultReporterMutex.Unlock()
	if rep != nil {
		if atomic.LoadInt32(&defaultRegistryInit) == 1 {
			copyTags := make(map[string]string, len(tags))
			for k, v := range tags {
				copyTags[k] = v
			}
			atomic.StorePointer(&rep.tags, unsafe.Pointer(&copyTags))
		}
	}
}
func (r *reporter) makeClient() (err error) {
	r.client = &http.Client{
		Timeout: 10 * time.Second,
	}
	return
}

func (r *reporter) run() {
	intervalTicker := time.NewTicker(r.interval)
	defer intervalTicker.Stop()
	d := r.interval - 5*time.Millisecond
	if d < 0 {
		d = 500 * time.Millisecond
	}
	memStatTicker := time.NewTicker(d)
	defer memStatTicker.Stop()
	for {
		select {
		case <-intervalTicker.C:
			if err := r.send(); err != nil {
				log.Printf("unable to send metrics to Falcon. err=%v", err)
			}
		case <-memStatTicker.C:
			metrics.CaptureDebugGCStatsOnce(r.reg)
			metrics.CaptureRuntimeMemStatsOnce(r.reg)
			currentGCPauseNs()
		}
	}
}

var (
	numGC         uint32             // NumGC指完成GC的次数
	pauseNsMetric = "GC.STW.PauseNs" // 不能再使用runtime前缀
)

// 统计当前时间段内的GC延迟情况
// 统计方式参考CaptureRuntimeMemStatsOnce()方法
func currentGCPauseNs() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats) // This takes 50-200us.
	i := numGC % uint32(len(memStats.PauseNs))
	ii := memStats.NumGC % uint32(len(memStats.PauseNs))
	if memStats.NumGC-numGC >= uint32(len(memStats.PauseNs)) {
		for i = 0; i < uint32(len(memStats.PauseNs)); i++ {
			TimerDuration(pauseNsMetric, time.Duration(memStats.PauseNs[i]))
		}
	} else {
		if i > ii {
			for ; i < uint32(len(memStats.PauseNs)); i++ {
				TimerDuration(pauseNsMetric, time.Duration(memStats.PauseNs[i]))
			}
			i = 0
		}
		for ; i < ii; i++ {
			TimerDuration(pauseNsMetric, time.Duration(memStats.PauseNs[i]))
		}
	}
	numGC = memStats.NumGC
}

type statPostData struct {
	Metric      string  `json:"metric"`
	Endpoint    string  `json:"endpoint"`
	Timestamp   int64   `json:"timestamp"`
	Value       float64 `json:"value"`
	Step        int     `json:"step"`
	ContentType string  `json:"counterType"`
	Tags        string  `json:"tags"`
}

// http://book.open-falcon.org/zh/usage/data-push.html

func (r *reporter) send() error {
	pts := make([]statPostData, 0)

	timerMetrics := make(map[*FieldMetadata]metrics.Timer)
	now := time.Now()
	nowDate := now.Format("2006-01-02 15:04:05")

	commentTable := newMetricTable([]string{"Name", "Code", "Comment"})
	countTable := newMetricTable([]string{"Date", "Name", "Tags", "Count"})
	gaugeTable := newMetricTable([]string{"Date", "Name", "Tags", "Gauge"})
	histogramTable := newMetricTable([]string{"Date", "Name", "Tags", "Total", "Max", "Min", "Avg", "Pt50", "Pt95", "Pt99", "Pt999", "Sum"})
	meterTable := newMetricTable([]string{"Date", "Name", "Tags", "Total", "Count", "Rate", "Rate1", "Rate5", "Rate15"})
	// nolint:lll
	timerTable := newMetricTable([]string{"Date", "Name", "Tags", "Total", fmt.Sprintf("Count(%s)", r.interval), "Rate1(qps)", "Max(ms)", "Min(ms)", "Avg(ms)", "Pt50(ms)", "Pt95(ms)", "Pt99(ms)", "Pt999(ms)", "Sum(s)", "Rate(%%)"})

	registryTags := *(*map[string]string)(atomic.LoadPointer(&r.tags))

	r.reg.Each(func(name string, i interface{}) {
		if !isMetricAccepted(name) {
			return
		}
		fieldMetaData := getFieldMetaDataFromString(name)
		tags := combineMaps(fieldMetaData.Tags, registryTags, eventTag, fieldMetaData.Name)
		tableTags := mapToString(fieldMetaData.Tags, TagCode)

		for code, comment := range fieldMetaData.Comment {
			commentTable.Append([]string{fieldMetaData.Name, strconv.Itoa(code), comment})
		}

		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			pts = append(pts, statPostData{
				Metric:      eventCount,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count()),
				Timestamp:   now.Unix(),
			})
			countTable.Append([]string{nowDate, fieldMetaData.Name, tableTags, strconv.FormatInt(ms.Count(), 10)})
		case metrics.Gauge:
			ms := metric.Snapshot()
			pts = append(pts, statPostData{
				Metric:      eventGauge,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Value()),
				Timestamp:   now.Unix(),
			})
			gaugeTable.Append([]string{nowDate, fieldMetaData.Name, tableTags, strconv.FormatInt(ms.Value(), 10)})
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			pts = append(pts, statPostData{
				Metric:      eventGauge,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.Value(),
				Timestamp:   now.Unix(),
			})
			gaugeTable.Append([]string{nowDate, fieldMetaData.Name, tableTags, floatToString(ms.Value())})
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			histogramTable.Append([]string{nowDate,
				fieldMetaData.Name,
				tableTags,
				strconv.FormatInt(ms.Count(), 10),
				strconv.FormatInt(ms.Max(), 10),
				strconv.FormatInt(ms.Min(), 10),
				floatToString(ms.Mean()),
				floatToString(ps[0]),
				floatToString(ps[2]),
				floatToString(ps[3]),
				floatToString(ps[4]),
				strconv.FormatInt(ms.Sum(), 10),
			})
			pts = append(pts, statPostData{
				Metric:      eventTotal,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Sum()),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventCount,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count()),
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventMax,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       float64(ms.Max()),
			// 	Timestamp:   now.Unix(),
			// })
			pts = append(pts, statPostData{
				Metric:      eventAvg,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.Mean(),
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventMin,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       float64(ms.Min()),
			// 	Timestamp:   now.Unix(),
			// })
			pts = append(pts, statPostData{
				Metric:      eventP50,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ps[0],
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventP75,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[1],
			// 	Timestamp:   now.Unix(),
			// })
			// pts = append(pts, statPostData{
			// 	Metric:      eventP95,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[2],
			// 	Timestamp:   now.Unix(),
			// })
			pts = append(pts, statPostData{
				Metric:      eventP99,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ps[3],
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventP999,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[4],
			// 	Timestamp:   now.Unix(),
			// })
			// pts = append(pts, statPostData{
			// 	Metric:      eventSum,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       float64(metric.Sum()),
			// 	Timestamp:   now.Unix(),
			// })
		case metrics.Meter:
			ms := metric.Snapshot()
			countMapKey := fieldMetaData.Name + "|" + tags
			oldCount := meterCountMap[countMapKey]
			meterTable.Append([]string{nowDate,
				fieldMetaData.Name,
				tableTags,
				strconv.FormatInt(ms.Count(), 10),
				strconv.FormatInt(ms.Count()-oldCount, 10),
				floatToString(ms.RateMean()),
				floatToString(ms.Rate1()),
				floatToString(ms.Rate5()),
				floatToString(ms.Rate15()),
			})
			meterCountMap[countMapKey] = ms.Count()
			pts = append(pts, statPostData{
				Metric:      eventTotal,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count()),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventCount,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count() - oldCount),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric: eventRate1, Tags: tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.Rate1(),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventRate,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.RateMean(),
				Timestamp:   now.Unix(),
			})
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			countMapKey := fieldMetaData.Name + "|" + tags + "-count"
			oldCount := eventCountMap[countMapKey]
			if _, ok := fieldMetaData.Tags[TagCode]; ok {
				timerMetrics[fieldMetaData] = ms
			} else {
				timerTable.Append([]string{
					nowDate,
					fieldMetaData.Name,
					tableTags,
					strconv.FormatInt(ms.Count(), 10),
					strconv.FormatInt(ms.Count()-oldCount, 10),
					floatToString(ms.Rate1()),
					floatToString(float64(ms.Max()) / 1e6),
					floatToString(float64(ms.Min()) / 1e6),
					floatToString(ms.Mean() / 1e6),
					floatToString(ps[0] / 1e6),
					floatToString(ps[2] / 1e6),
					floatToString(ps[3] / 1e6),
					floatToString(ps[4] / 1e6),
					floatToString(float64(ms.Sum()) / 1e9),
					floatToString(float64(100)),
				})

			}
			eventCountMap[countMapKey] = ms.Count()
			pts = append(pts, statPostData{
				Metric:      eventCodeCount,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count() - oldCount),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventCount,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Count()),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventTimeMax,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Max() / 1e6),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventCodeAvgtime,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.Mean() / 1e6,
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventTimeMin,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(ms.Min() / 1e6),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventTimeP50,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ps[0] / 1e6,
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventTimeP75,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[1] / 1e6,
			// 	Timestamp:   now.Unix(),
			// })
			// pts = append(pts, statPostData{
			// 	Metric:      eventTimeP95,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[2] / 1e6,
			// 	Timestamp:   now.Unix(),
			// })
			pts = append(pts, statPostData{
				Metric:      eventTimeP99,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ps[3] / 1e6,
				Timestamp:   now.Unix(),
			})
			// pts = append(pts, statPostData{
			// 	Metric:      eventTimeP999,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       ps[4] / 1e6,
			// 	Timestamp:   now.Unix(),
			// })
			// pts = append(pts, statPostData{
			// 	Metric:      eventTimeSum,
			// 	Tags:        tags,
			// 	Endpoint:    localEndPoint,
			// 	Step:        int(r.interval.Seconds()),
			// 	ContentType: "GAUGE",
			// 	Value:       float64(metric.Sum() / 1e9),
			// 	Timestamp:   now.Unix(),
			// })
			pts = append(pts, statPostData{
				Metric:      eventRate1,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.Rate1(),
				Timestamp:   now.Unix(),
			})
			pts = append(pts, statPostData{
				Metric:      eventRate,
				Tags:        tags,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       ms.RateMean(),
				Timestamp:   now.Unix(),
			})
		}
	})

	var getTagString = func(meta *FieldMetadata, except string) string {
		return meta.Name + "|" + mapToString(meta.Tags, except)
	}

	// 拿出之前的数量
	oldCountMap := make(map[string]int64)
	for meta, ms := range timerMetrics {
		totalTagsString := getTagString(meta, TagCode)
		codeTagsString := getTagString(meta, "")
		oldCountMap[totalTagsString] = eventCountMap[totalTagsString]
		oldCountMap[codeTagsString] = eventCountMap[codeTagsString]
		eventCountMap[codeTagsString] = ms.Count()
	}

	// 找出存在code=0的totalTagsString && 找出successCodeMap转换之后的code=0的totalTagsString
	code0TotalTagsMap := make(map[string]bool)
	// 找出存在code=0的totalTagsString
	realCode0TotalTagsMap := make(map[string]bool)

	codeConvertMap := make(map[string]int64)
	// 计算没有code的总量
	for meta, ms := range timerMetrics {
		totalTagsString := getTagString(meta, TagCode)
		codeTagsString := getTagString(meta, "")
		eventCountMap[totalTagsString] = eventCountMap[totalTagsString] + (ms.Count() - oldCountMap[codeTagsString])

		if strings.Contains(codeTagsString, "code=0") {
			code0TotalTagsMap[totalTagsString] = true
			realCode0TotalTagsMap[totalTagsString] = true
		}

		if tagValue, ok := meta.Tags[TagCode]; ok {
			if code, err := strconv.Atoi(tagValue); err == nil {
				successCodeMapMutex.Lock()
				_, load := successCodeMap.Load(code)
				successCodeMapMutex.Unlock()
				if load {
					// 标记success code metric name
					code0TotalTagsMap[totalTagsString] = true

					// copy tags map
					tagsNew := make(map[string]string)
					for k, v := range meta.Tags {
						if k == TagCode {
							tagsNew[k] = "0"
						} else {
							tagsNew[k] = v
						}
					}
					// 把转化出的值暂存到converted里面
					convertedCodeTagsString := meta.Name + "|" + mapToString(tagsNew, "")
					codeConvertMap[convertedCodeTagsString] += ms.Count() - oldCountMap[codeTagsString]
				}
			}
		}
	}

	// code=0 tag
	appendCode0TagMap := make(map[string]bool)
	// success code tag
	appendSuccessCodeTagMap := make(map[string]bool)

	noCodeCounterMap := make(map[string]bool)
	for meta, ms := range timerMetrics {
		totalTagsString := getTagString(meta, TagCode)
		codeTagsString := getTagString(meta, "")

		totalDiff := eventCountMap[totalTagsString] - oldCountMap[totalTagsString]
		codeDiff := eventCountMap[codeTagsString] - oldCountMap[codeTagsString]

		codeRatio := 0.0
		if totalDiff != 0 {
			codeRatio = 100.0 * float64(codeDiff) / float64(totalDiff)
		} else if meta.Tags[TagCode] == "0" {
			codeRatio = 100.0
		}
		pts = append(pts, statPostData{
			Metric:      eventCode0Ratio,
			Tags:        combineMaps(meta.Tags, registryTags, eventTag, meta.Name),
			Endpoint:    localEndPoint,
			Step:        int(r.interval.Seconds()),
			ContentType: "GAUGE",
			Value:       codeRatio,
			Timestamp:   now.Unix(),
		})

		// 上报存在code但是不存在code=0(包含convert之后的code=0)的成功率
		_, ok1 := code0TotalTagsMap[totalTagsString]
		_, ok2 := appendCode0TagMap[totalTagsString] // 用于标记重复上报
		if !ok1 && !ok2 {
			// 如果存在code码，上报code=0 为0%
			// 如果不存在任何code码，上报code=0 为100%
			virtualCode0Val := 0.0
			if totalDiff == 0 {
				virtualCode0Val = 100.0
			}

			appendCode0TagMap[totalTagsString] = true
			code0MetaTags := getCode0Tags(meta.Tags)

			pts = append(pts, statPostData{
				Metric:      eventCodeCount,
				Tags:        combineMaps(code0MetaTags, registryTags, eventTag, meta.Name),
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       0.0,
				Timestamp:   now.Unix(),
			})

			pts = append(pts, statPostData{
				Metric:      eventCodeRatio,
				Tags:        combineMaps(code0MetaTags, registryTags, eventTag, meta.Name),
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       virtualCode0Val,
				Timestamp:   now.Unix(),
			})

			timerTable.Append([]string{
				nowDate,
				meta.Name,
				mapToString(code0MetaTags, ""),
				strconv.FormatInt(0, 10),
				strconv.FormatInt(0, 10),
				floatToString(0),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e6),
				floatToString(0 / 1e9),
				floatToString(virtualCode0Val),
			})
		}

		// 只有存在code=0，才会进入下面convert_map判断的逻辑
		if cnt, ok := codeConvertMap[codeTagsString]; ok {
			convertedCodeDiff := eventCountMap[codeTagsString] - oldCountMap[codeTagsString] + cnt
			convertedCodeRatio := 0.0
			if totalDiff != 0 {
				convertedCodeRatio = 100.0 * float64(convertedCodeDiff) / float64(totalDiff)
			} else if meta.Tags[TagCode] == "0" {
				convertedCodeRatio = 100.0
			}
			pts = append(pts, statPostData{
				Metric:      eventCodeRatio,
				Tags:        combineMaps(meta.Tags, registryTags, eventTag, meta.Name),
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       convertedCodeRatio,
				Timestamp:   now.Unix(),
			})
		} else {
			// 如果不存在code=0, 但是存在success code
			ok1 := isSuccessCode(meta.Tags[TagCode])           // success code
			_, ok2 := realCode0TotalTagsMap[totalTagsString]   // 不在code=0的map中，即代表这一组tags不存在code=0
			_, ok3 := appendSuccessCodeTagMap[totalTagsString] // 并且之前未上报过
			if ok1 && !ok2 && !ok3 {
				appendSuccessCodeTagMap[totalTagsString] = true
				successCodeTagsString := getSuccessCodeTagsString(meta.Name, meta.Tags)

				code0MetaTags := getCode0Tags(meta.Tags)

				convertedCodeDiff := codeConvertMap[successCodeTagsString]

				convertedCodeRatio := 100.0
				if totalDiff != 0 {
					convertedCodeRatio = 100.0 * float64(convertedCodeDiff) / float64(totalDiff)
				}
				pts = append(pts, statPostData{
					Metric:      eventCodeRatio,
					Tags:        combineMaps(code0MetaTags, registryTags, eventTag, meta.Name),
					Endpoint:    localEndPoint,
					Step:        int(r.interval.Seconds()),
					ContentType: "GAUGE",
					Value:       convertedCodeRatio,
					Timestamp:   now.Unix(),
				})
			}

			// 上报原code的metric信息
			pts = append(pts, statPostData{
				Metric:      eventCodeRatio,
				Tags:        combineMaps(meta.Tags, registryTags, eventTag, meta.Name),
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       codeRatio,
				Timestamp:   now.Unix(),
			})
		}

		// 上传没有code的count总量
		noCodeCounter := combineMapsWithExceptTag(meta.Tags, registryTags, TagCode, eventTag, meta.Name)
		if _, ok := noCodeCounterMap[noCodeCounter]; !ok {
			pts = append(pts, statPostData{
				Metric:      eventTotal,
				Tags:        noCodeCounter,
				Endpoint:    localEndPoint,
				Step:        int(r.interval.Seconds()),
				ContentType: "GAUGE",
				Value:       float64(totalDiff),
				Timestamp:   now.Unix(),
			})
			noCodeCounterMap[noCodeCounter] = true
		}

		ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
		timerTable.Append([]string{
			nowDate,
			meta.Name,
			mapToString(meta.Tags, ""),
			strconv.FormatInt(ms.Count(), 10),
			strconv.FormatInt(codeDiff, 10),
			floatToString(ms.Rate1()),
			floatToString(float64(ms.Max()) / 1e6),
			floatToString(float64(ms.Min()) / 1e6),
			floatToString(ms.Mean() / 1e6),
			floatToString(ps[0] / 1e6),
			floatToString(ps[2] / 1e6),
			floatToString(ps[3] / 1e6),
			floatToString(ps[4] / 1e6),
			floatToString(float64(ms.Sum()) / 1e9),
			floatToString(codeRatio),
		})
	}
	serverTagsString := combineMaps(registryTags, nil)

	if commentTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Comment,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(commentTable.String())
	}
	if countTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Count Data,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(countTable.String())
	}
	if gaugeTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Gauge Data,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(gaugeTable.String())

	}
	if meterTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Meter Data,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(meterTable.String())
	}
	if histogramTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Histogram Data,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(histogramTable.String())
	}
	if timerTable.Size() > 0 {
		dataLogger.Printf("Server[%s] (Timer Data,  Pid %d, LocalIP %s, Date %s) last %d seconds Statistic Info\n", serverTagsString, runPid, localEndPoint, nowDate, int(r.interval.Seconds()))
		dataLogger.Printf(timerTable.String())
	}
	if roll != nil {
		_ = roll.Sync()
	}
	data := bytes.NewBuffer(nil)
	_ = json.NewEncoder(data).Encode(pts)
	rsp, err := r.client.Post(r.url, "application/json", data)
	if err != nil {
		return err
	}
	_ = rsp.Body.Close()
	return err
}

func combineMapsWithExceptTag(a, b map[string]string, exceptTag string, kvs ...string) string {
	newM := make(map[string]string, len(kvs)/2+len(b)+len(a))
	for k, v := range a {
		newM[k] = v
	}
	for k, v := range b {
		newM[k] = v
	}
	if len(kvs)%2 == 0 {
		for i := 0; i < len(kvs); i = i + 2 {
			newM[kvs[i]] = kvs[i+1]
		}
	}
	return mapToString(newM, exceptTag)
}

func combineMaps(a, b map[string]string, kvs ...string) string {
	return combineMapsWithExceptTag(a, b, "", kvs...)
}

func isSuccessCode(codeVal string) bool {
	if code, err := strconv.Atoi(codeVal); err == nil {
		successCodeMapMutex.Lock()
		defer successCodeMapMutex.Unlock()
		if _, ok := successCodeMap.Load(code); ok {
			return true
		}
	}
	return false
}

func getSuccessCodeTagsString(name string, tags map[string]string) string {
	successCodeMetaTags := make(map[string]string)
	for k, v := range tags {
		successCodeMetaTags[k] = v
	}
	successCodeMetaTags[TagCode] = "0"
	return name + "|" + mapToString(successCodeMetaTags, "")
}

func getCode0Tags(tags map[string]string) map[string]string {
	code0MetaTags := make(map[string]string)
	for k, v := range tags {
		code0MetaTags[k] = v
	}
	code0MetaTags[TagCode] = "0"
	return code0MetaTags
}
