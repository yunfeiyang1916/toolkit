package metrics

import (
	"bytes"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type metricTable struct {
	buf    *bytes.Buffer
	table  *tablewriter.Table
	data   [][]string
	header []string
}

type tableDataSlice [][]string

func (t tableDataSlice) Len() int {
	return len(t)
}

func (t tableDataSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tableDataSlice) Less(i, j int) bool {
	l := len(t[i])
	for m := 0; m < l; m++ {
		n := strings.Compare(t[i][m], t[j][m])
		if n < 0 {
			return true
		} else if n > 0 {
			return false
		}
	}
	return false
}

func newMetricTable(h []string) *metricTable {
	t := &metricTable{
		buf:    &bytes.Buffer{},
		header: h,
	}
	t.table = tablewriter.NewWriter(t.buf)
	t.table.SetHeader(t.header)
	t.data = make([][]string, 0)
	return t
}

var acceptedRuntimeMetric = make(map[string]bool)

func init() {
	acceptedRuntimeMetric["runtime.MemStats.Alloc"] = true
	acceptedRuntimeMetric["runtime.MemStats.Sys"] = true
	acceptedRuntimeMetric["runtime.MemStats.Mallocs"] = true
	acceptedRuntimeMetric["runtime.MemStats.Frees"] = true
	acceptedRuntimeMetric["runtime.MemStats.HeapObjects"] = true
	acceptedRuntimeMetric["runtime.MemStats.PauseTotalNs"] = true
	acceptedRuntimeMetric["runtime.MemStats.PauseNs"] = true
	acceptedRuntimeMetric["runtime.NumCgoCall"] = true
	acceptedRuntimeMetric["runtime.NumGoroutine"] = true
	acceptedRuntimeMetric["runtime.NumThread"] = true
	acceptedRuntimeMetric["debug.GCStats.NumGC"] = true
	acceptedRuntimeMetric["debug.GCStats.PauseTotal"] = true
}

func isMetricAccepted(name string) bool {
	if strings.HasPrefix(name, "runtime.") || strings.HasPrefix(name, "debug.") {
		return acceptedRuntimeMetric[name]
	}
	return true
}

func (t *metricTable) Append(line []string) {
	t.data = append(t.data, line)
}

func (t *metricTable) Size() int {
	return len(t.data)
}

func (t *metricTable) String() string {
	sort.Sort(tableDataSlice(t.data))
	t.table.AppendBulk(t.data)
	t.table.Render()
	return t.buf.String()
}
