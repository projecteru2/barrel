package ipam

import (
	"context"

	"github.com/docker/go-plugins-helpers/network"
	"github.com/juju/errors"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"

	dockerTypes "github.com/docker/docker/api/types"
	dockerNetworkTypes "github.com/docker/docker/api/types/network"

	dockerClient "github.com/docker/docker/client"
	logutils "github.com/projectcalico/libnetwork-plugin/utils/log"
	calicoDriver "github.com/projecteru2/barrel/ipam/calico"
	"github.com/projecteru2/barrel/types"
)

// NetDriver .
type NetDriver struct {
	calNetDriver calicoDriver.NetworkDriver
	dockerCli    *dockerClient.Client
	ipam         types.ReservedAddressManager
}

// GetCapabilities .
func (driver *NetDriver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	return driver.calNetDriver.GetCapabilities()
}

// AllocateNetwork .
func (driver *NetDriver) AllocateNetwork(request *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	return driver.calNetDriver.AllocateNetwork(request)
}

// FreeNetwork is used for swarm-mode support in remote plugins, which
// Calico's libnetwork-plugin doesn't currently support.
func (driver *NetDriver) FreeNetwork(request *network.FreeNetworkRequest) error {
	return driver.calNetDriver.FreeNetwork(request)
}

// CreateNetwork .
func (driver *NetDriver) CreateNetwork(request *network.CreateNetworkRequest) error {
	return driver.calNetDriver.CreateNetwork(request)
}

// DeleteNetwork .
func (driver *NetDriver) DeleteNetwork(request *network.DeleteNetworkRequest) error {
	return driver.calNetDriver.DeleteNetwork(request)
}

// CreateEndpoint .
func (driver *NetDriver) CreateEndpoint(request *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	return driver.calNetDriver.CreateEndpoint(request)
}

// DeleteEndpoint .
func (driver *NetDriver) DeleteEndpoint(request *network.DeleteEndpointRequest) error {
	return driver.calNetDriver.DeleteEndpoint(request)
}

// EndpointInfo .
func (driver *NetDriver) EndpointInfo(request *network.InfoRequest) (*network.InfoResponse, error) {
	return driver.calNetDriver.EndpointInfo(request)
}

// Join .
func (driver *NetDriver) Join(request *network.JoinRequest) (*network.JoinResponse, error) {
	return driver.calNetDriver.Join(request)
}

// Leave .
func (driver *NetDriver) Leave(request *network.LeaveRequest) error {
	logutils.JSONMessage("Leave response", request)
	var (
		container        dockerTypes.Container
		endpointSettings *dockerNetworkTypes.EndpointSettings
		pool             *api.IPPool
		err              error
	)
	if container, endpointSettings, err = driver.findDockerContainerByEndpointID(request.EndpointID); err != nil {
		return err
	}

	if pool, err = driver.calNetDriver.FindPoolByNetworkID(endpointSettings.NetworkID); err != nil {
		return err
	}

	if containerHasFixedIPLabel(container) {
		if err = driver.ipam.ReserveAddressForContainer(
			container.ID,
			types.Address{
				PoolID:  pool.Name,
				Address: endpointSettings.IPAddress,
			},
		); err != nil {
			// we move on when reserve is failed
			log.Errorln(err)
		}
	}
	return driver.calNetDriver.Leave(request)
}

func (driver *NetDriver) findDockerContainerByEndpointID(endpointID string) (dockerTypes.Container, *dockerNetworkTypes.EndpointSettings, error) {
	containers, err := driver.dockerCli.ContainerList(context.Background(), dockerTypes.ContainerListOptions{})
	if err != nil {
		log.Errorf("dockerCli ContainerList Error, %v", err)
		return dockerTypes.Container{}, nil, err
	}
	for _, container := range containers {
		for _, network := range container.NetworkSettings.Networks {
			if endpointID == network.EndpointID {
				return container, network, nil
			}
		}
	}
	return dockerTypes.Container{}, nil, errors.Errorf("find no container with endpintID = %s", endpointID)
}

// DiscoverNew .
func (driver *NetDriver) DiscoverNew(request *network.DiscoveryNotification) error {
	return driver.calNetDriver.DiscoverNew(request)
}

// DiscoverDelete .
func (driver *NetDriver) DiscoverDelete(request *network.DiscoveryNotification) error {
	return driver.calNetDriver.DiscoverDelete(request)
}

// ProgramExternalConnectivity .
func (driver *NetDriver) ProgramExternalConnectivity(request *network.ProgramExternalConnectivityRequest) error {
	return driver.calNetDriver.ProgramExternalConnectivity(request)
}

// RevokeExternalConnectivity .
func (driver *NetDriver) RevokeExternalConnectivity(request *network.RevokeExternalConnectivityRequest) error {
	return driver.calNetDriver.RevokeExternalConnectivity(request)
}
