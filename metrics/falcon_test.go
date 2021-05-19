package metrics

import (
	"fmt"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

func TestFalcon(t *testing.T) {
	go Falcon(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
	)

	// recently added - support for tags per metric

	fieldMetadata := FieldMetadata{Name: "request", Tags: map[string]string{"status-code": "200", "method": "GET", "path": "/serviceinfo/info"}}
	// tag metadata is encoded into the existing 'name' field for posting to falcon, as json
	meter := metrics.NewMeter()
	metrics.Register(fieldMetadata.String(), meter)
	meter.Mark(64)
	// metrics.DefaultRegistry.GetOrRegister(fieldMetadata.String(), meter)

	// tag metadata is encoded into the existing 'name' field for posting to falcon, as json
	c := metrics.NewCounter()
	metrics.Register((&FieldMetadata{Name: "foo-count", Tags: nil}).String(), c)
	c.Inc(47)

	g := metrics.NewGauge()
	metrics.Register((&FieldMetadata{Name: "bar-gauge", Tags: nil}).String(), g)
	g.Update(47)

	// g := metrics.NewRegisteredFunctionalGauge("cache-evictions", r, func() int64 { return cache.getEvictionsCount() })

	s := metrics.NewExpDecaySample(1028, 0.015) // or metrics.NewUniformSample(1028)
	h := metrics.NewHistogram(s)
	metrics.Register((&FieldMetadata{Name: "bar-his", Tags: nil}).String(), h)
	h.Update(47)

	tm := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "origin-bang-timer", Tags: nil}).String(), tm)
	tm.Time(func() {})
	tm.Update(47)
	time.Sleep(2 * time.Second)

	tags := map[string]string{"code": "500"}

	// ============== name="bang-timer mode:add success code" ======================

	cm := map[int]int{403: 1}
	AddSuccessCode(cm)

	tm2 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "499"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "400"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "403"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	// ============== name="pang-timer mode:normal" ======================
	tags["code"] = "500"
	tm3 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "pang-timer", Tags: tags}).String(), tm3)
	tm3.Time(func() {})
	tm3.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "0"
	metrics.Register((&FieldMetadata{Name: "pang-timer", Tags: tags}).String(), tm3)
	tm3.Time(func() {})
	tm3.Update(47)
	time.Sleep(2 * time.Second)

	// ============== name="code0-timer mode:multi tags" ======================
	tags["code"] = "500"
	tags["peer"] = "queen.sociallink.voicetime"
	tm4 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "code0-timer", Tags: tags}).String(), tm4)
	tm4.Time(func() {})
	tm4.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "700"
	tags["peer"] = "queen.oper.parttime_backend"
	metrics.Register((&FieldMetadata{Name: "code0-timer", Tags: tags}).String(), tm4)
	tm4.Time(func() {})
	tm4.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "500"
	tags["peer"] = "queen.oper.parttime_backend"
	metrics.Register((&FieldMetadata{Name: "code0-timer", Tags: tags}).String(), tm4)
	tm4.Time(func() {})
	tm4.Update(47)
	time.Sleep(2 * time.Second)
}

func TestFieldMeata(t *testing.T) {
	fm := FieldMetadata{
		Name: "name",
		Tags: map[string]string{"a": "b", "c": "d", "e": "f"},
	}
	fmt.Printf("origin %#v, st:%q, after %#v\n", fm, fm.String(), getFieldMetaDataFromString(fm.String()))
}

func TestSuccessCodeWithCode0ToFalcon(t *testing.T) {

	go Falcon(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
	)

	tags := map[string]string{"code": "500"}

	// ============== name="bang-timer mode:add success code" ======================

	cm := map[int]int{403: 1}
	AddSuccessCode(cm)

	tm2 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "499"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "400"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "403"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "0"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)
}

func TestSuccessCodeWithCode0ToFalcon2(t *testing.T) {

	go Falcon(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
	)

	tags := map[string]string{"code": "500"}

	// ============== name="bang-timer mode:add success code" ======================

	cm := map[int]int{403: 1}
	AddSuccessCode(cm)

	tm2 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "0"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "499"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "400"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "403"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)
}

func TestSuccessCodeWithoutCode0ToFalcon(t *testing.T) {

	go Falcon(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
	)

	tags := map[string]string{"code": "500"}

	// ============== name="bang-timer mode:add success code" ======================

	cm := map[int]int{403: 1}
	AddSuccessCode(cm)

	tm2 := metrics.NewTimer()
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "499"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "400"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "403"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)

	tags["code"] = "600"
	metrics.Register((&FieldMetadata{Name: "bang-timer", Tags: tags}).String(), tm2)
	tm2.Time(func() {})
	tm2.Update(47)
	time.Sleep(2 * time.Second)
}
