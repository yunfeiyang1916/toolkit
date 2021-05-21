package registry

import (
	"sort"
	"strings"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
)

// Cluster is service all health endpoints
type Cluster struct {
	Name      string
	Endpoints []Endpoint
}

// Endpoint describe one health cluster instance
type Endpoint struct {
	ID   string
	Addr string
	Port int
	Tags []string
}

type Backend interface {
	// Register registers a service in the registry.
	Register(cfg *config.Register) error

	// Deregister removes the service registration.
	Deregister(cfg *config.Register) error

	// ReadManual returns the current manual overrides and
	// their version as seen by the registry.
	ReadManual(KVPath string) (value string, version uint64, err error)

	// WriteManual writes the new value to the registry if the
	// version of the stored document still matchhes version.
	WriteManual(KVPath, value string, version uint64) (ok bool, err error)

	// WatchServices watches the registry for changes in service
	// registration and health and pushes them if there is a difference.
	WatchServices(name string, status []string, dc string) chan []*Cluster

	// WatchManual watches the registry for changes in the manual
	// overrides and pushes them if there is a difference.
	WatchManual(KVPath string) chan string

	// WatchPrefixManual watches the registry for changes in the manual
	// overrides and pushes them if there is a difference.
	WatchPrefixManual(KVPath string) chan map[string]string
}

var Default Backend

// AddEnvTag add default env tag
func (cluster *Cluster) AddEnvTag() {
	for i, end := range cluster.Endpoints {
		found := false
		for j, s := range end.Tags {
			if strings.HasPrefix(s, "env=") {
				ss := strings.SplitN(s, "=", 2)
				if len(ss) == 2 && len(ss[1]) == 0 {
					end.Tags[j] = "env=online"
				}
				found = true
				break
			}
		}
		if !found {
			end.Tags = append(end.Tags, "env=online")
		}

		sort.Strings(end.Tags)

		cluster.Endpoints[i] = Endpoint{
			ID:   end.ID,
			Addr: end.Addr,
			Port: end.Port,
			Tags: end.Tags,
		}
	}
}
