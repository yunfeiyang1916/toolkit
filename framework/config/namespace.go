package config

import (
	"path/filepath"

	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/consul"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/file"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/memory"
)

var ConsulAddr = "127.0.0.1:8500"

type Namespace struct {
	namespace string
}

func NewNamespace(namespace string) *Namespace {
	return &Namespace{namespace}
}

func NewNamespaceD() *Namespace {
	return &Namespace{"default"}
}

func (m *Namespace) Get(resource string) Config {
	return m.GetD(resource, "", true)
}

func (m *Namespace) With(resource string) *Namespace {
	return NewNamespace(filepath.Join(m.namespace, resource))
}

func (m *Namespace) GetD(resource, filename string, remoteFirst bool) Config {
	remote := consul.NewSource(
		consul.WithAddress(ConsulAddr),
		consul.WithAbsPath(filepath.Join(m.namespace, resource)),
		consul.UsePrefix(false),
		source.WithEncoder(TomlEncoder()),
	)

	local := file.NewSource(source.WithEncoder(TomlEncoder()), file.WithPath(filename))

	if remoteFirst {
		return New(WithSource(remote))
	}
	return New(WithSource(local))
}

func (m *Namespace) GetMemoryD(mem []byte) Config {
	return New(
		WithSource(memory.NewSource(memory.WithDataToml(mem))),
	)
}
