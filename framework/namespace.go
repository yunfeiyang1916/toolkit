package framework

import (
	"os"
	"path"
	"sync"

	ns "github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

var (
	GlobalNamespace    = newNamespace()
	loadtestRemotePath = "/app/loadtest/config.toml"
)

type namespace struct {
	m sync.Map
}

func newNamespace() *namespace {
	return &namespace{}
}

func (n *namespace) Add(namespace string, opts ...Option) {
	if _, ok := n.m.Load(namespace); ok {
		return
	}
	ns.AddNamespaceKey(namespace)
	opts = append(opts, Namespace(namespace))
	d := New()
	d.Init(opts...)
	n.m.LoadOrStore(namespace, d)
}

// param namespace represents the app name
func (n *namespace) Get(namespace string) *Framework {
	d, _ := n.m.Load(namespace)
	if d == nil {
		return nil
	}
	return d.(*Framework)
}

func (n *namespace) All() []*Framework {
	var result []*Framework
	n.m.Range(func(key, value interface{}) bool {
		result = append(result, value.(*Framework))
		return true
	})
	return result
}

func (n *namespace) Ctxs() []context.Context {
	var result []context.Context
	n.m.Range(func(key, value interface{}) bool {
		result = append(result, WithAPPKey(context.TODO(), value.(*Framework).Namespace))
		return true
	})
	return result
}

//nolint:unused
func (n *namespace) update(namespace string, opts ...Option) {
	val, ok := n.m.Load(namespace)
	if ok {
		d := val.(*Framework)
		_ = d.shutdown()
		n.m.Delete(namespace)
	}
	opts = append(opts, Namespace(namespace))
	d := New()
	d.Init(opts...)
	n.m.LoadOrStore(namespace, d)
	logging.GenLogf("%s namespace update...", namespace)
}

//nolint:unused
func (n *namespace) del(namespace string) {
	val, ok := n.m.Load(namespace)
	if ok {
		d := val.(*Framework)
		_ = d.shutdown()
		n.m.Delete(namespace)
	}
	logging.GenLogf("%s namespace del...", namespace)
}

type appKeyType struct{}

var appkey = appKeyType{}

func WithAPPKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, appkey, key)
}

func FromContext(ctx context.Context) (*Framework, bool) {
	key := ctx.Value(appkey)
	if key == nil {
		return nil, false
	}
	if d := GlobalNamespace.Get(key.(string)); d != nil {
		return d, true
	}
	return nil, false
}

func init() {
	logging.RegisteCtx(func(ctx context.Context) (string, string) {
		return "namespace", ns.GetNamespace(ctx)
	})
}

func InitNamespace(dir string) {
	fd, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	fds, err := fd.Readdir(-1)
	if err != nil {
		panic(err)
	}
	for _, fileInfo := range fds {
		if !fileInfo.IsDir() {
			continue
		}
		GlobalNamespace.Add(
			fileInfo.Name(),
			// app name
			Namespace(fileInfo.Name()),
			NamespaceDir(dir),
			ConfigPath(path.Join(dir, fileInfo.Name(), "config.toml")),
			App(Default.App),
			Name(Default.Name),
			Version(Default.Version),
			Deps(Default.Deps),
		)
	}
}
