package fixedip

import (
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/projectcalico/libcalico-go/lib/clientv3"

	calicoDriver "github.com/projecteru2/barrel/driver/calico"
	"github.com/projecteru2/barrel/vessel"
)

// Driver .
type Driver struct {
	calicoDriver.Driver
	agent vessel.CNMAgent
}

// NewDriver .
func NewDriver(
	client clientv3.Interface,
	dockerCli *dockerClient.Client,
	agent vessel.CNMAgent,
	hostname string,
) Driver {
	return Driver{
		Driver: calicoDriver.NewDriver(client, dockerCli, hostname),
		agent:  agent,
	}
}

// CreateEndpoint .
func (driver Driver) CreateEndpoint(request *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	resp, err := driver.Driver.CreateEndpoint(request)
	if err == nil && driver.agent != nil {
		driver.agent.NotifyEndpointCreated(request.NetworkID, request.EndpointID)
	}
	return resp, err
}

// DeleteEndpoint .
func (driver Driver) DeleteEndpoint(request *network.DeleteEndpointRequest) error {
	if err := driver.Driver.DeleteEndpoint(request); err != nil {
		return err
	}
	if driver.agent != nil {
		driver.agent.NotifyEndpointRemoved(request.NetworkID, request.EndpointID)
	}
	return nil
}
