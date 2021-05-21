package consul

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/logging"
)

// register keeps a service registered in consul.
//
// When a value is sent in the dereg channel the service is deregistered from
// consul. To wait for completion the caller should read the next value from
// the dereg channel.
//
//    dereg <- true // trigger deregistration
//    <-dereg       // wait for completion
//
func register(logger *logging.Logger, c *api.Client, service *api.AgentServiceRegistration, tagsOverride <-chan []string) (dereg chan bool) {
	var serviceID string

	registered := func() bool {
		if serviceID == "" {
			existService, _ := c.Agent().Services()
			for _, s := range existService {
				s := s
				if s.Address == service.Address && s.Port == service.Port && s.Service == service.Name && reflect.DeepEqual(s.Tags, service.Tags) {
					serviceID = s.ID
					return true
				}
			}
			return false
		}
		services, err := c.Agent().Services()
		if err != nil {
			logger.Errorf("consul: Cannot get service list. %s", err)
			return false
		}
		return services[serviceID] != nil
	}

	register := func() {
		if err := c.Agent().ServiceRegister(service); err != nil {
			logger.Errorf("consul: Cannot register service %s in consul. %s", service.Name, err)
			return
		}

		logger.Infof("consul: Registered service %s with id %q, address %s, tags %q, http check %q, tcp check %q", service.Name, service.ID, service.Address, strings.Join(service.Tags, ","), service.Check.HTTP, service.Check.TCP)
		serviceID = service.ID
	}

	deregister := func() {
		err := c.Agent().ServiceDeregister(serviceID)
		logger.Infof("consul: Deregistering service %s with id %s, err %v", service.Name, serviceID, err)
	}

	dereg = make(chan bool)
	go func() {
		register()
		for {
			select {
			case <-dereg:
				deregister()
				dereg <- true
				return
			case <-time.After(10 * time.Second):
				if !registered() {
					register()
				}
			case tags := <-tagsOverride:
				service.Tags = tags
				register()
			}
		}
	}()
	return dereg
}

func serviceRegistration(cfg *config.Register, tags []string) (*api.AgentServiceRegistration, error) {
	serviceID := fmt.Sprintf("%s-%s:%d", cfg.ServiceName, cfg.ServiceAddr, cfg.ServicePort)
	checker := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", cfg.DeregisterCriticalServiceAfterSec),
		Interval:                       fmt.Sprintf("%dms", cfg.ServiceCheckIntervalMs),
		Timeout:                        fmt.Sprintf("%dms", cfg.ServiceCheckTimeoutMs),
		TCP:                            fmt.Sprintf("%s:%d", cfg.ServiceAddr, cfg.ServicePort),
	}
	if len(cfg.ServiceCheckDSN) != 0 {
		checkDSNURL, err := url.Parse(cfg.ServiceCheckDSN)
		if err != nil {
			return nil, err
		}
		if checkDSNURL.Scheme == "http" {
			checker.HTTP = cfg.ServiceCheckDSN
		} else if checkDSNURL.Scheme == "tcp" {
			checker.TCP = checkDSNURL.Host
		}
	}

	service := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    cfg.ServiceName,
		Address: cfg.ServiceAddr,
		Port:    cfg.ServicePort,
		Tags:    tags,
		Check:   checker,
	}

	return service, nil
}
