package sd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
)

func RegistryKVPath(name, path string) (string, error) {
	namespace := strings.Split(name, ".")[0]
	if len(namespace) == 0 {
		return "", fmt.Errorf("wrong sdname %s", name)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = fmt.Sprintf("%s%s/%s%s", "/service_config/", namespace, name, path)
	return path, nil
}

func GetDatacenter(consulAddr string) (string, error) {
	c, err := api.NewClient(&api.Config{Address: consulAddr, Scheme: "http"})
	if err != nil {
		return "", err
	}

	self, err := c.Agent().Self()
	if err != nil {
		return "", err
	}

	serviceConfig, ok := self["Config"]
	if !ok {
		return "", errors.New("consul: self.Config not found")
	}

	dc, ok := serviceConfig["Datacenter"].(string)
	if !ok {
		return "", errors.New("consul: self.Datacenter not found")
	}

	return dc, nil
}
