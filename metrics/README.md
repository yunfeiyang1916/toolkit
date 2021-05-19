# go-metrics-falcon

This is a reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library which will post the metrics to Falcon

## Note

This is only compatible with Go 1.7+.

## Usage

```go
import "git.inke.cn/tpc/inf/metrics"

go metrics.New(
    metrics.DefaultRegistry, // metrics registry
    time.Second * 60,        // interval
)


fieldMetadata := falcon.FieldMetadata{Name: "request", Tags: map[string]string{"status-code": strconv.Itoa(rw.StatusCode()), "method": req.Method, "path": uriPath}}
// tag metadata is encoded into the existing 'name' field for posting to influx, as json
meter := metrics.NewMeter()
//registry.GetOrRegister(fieldMetadata.String(), meter)
```

## License
