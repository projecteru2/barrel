package calicoplus

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
	"github.com/projecteru2/barrel/driver"
	calicoDriver "github.com/projecteru2/barrel/driver/calicoplus/calico"
	"github.com/projecteru2/barrel/types"
)

type networkDriver struct {
	calNetDriver calicoDriver.NetworkDriver
	dockerCli    *dockerClient.Client
	ipam         driver.ReservedAddressManager
}

// GetCapabilities .
func (driver *networkDriver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	return driver.calNetDriver.GetCapabilities()
}

// AllocateNetwork .
func (driver *networkDriver) AllocateNetwork(request *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	return driver.calNetDriver.AllocateNetwork(request)
}

// FreeNetwork is used for swarm-mode support in remote plugins, which
// Calico's libnetwork-plugin doesn't currently support.
func (driver *networkDriver) FreeNetwork(request *network.FreeNetworkRequest) error {
	return driver.calNetDriver.FreeNetwork(request)
}

// CreateNetwork .
func (driver *networkDriver) CreateNetwork(request *network.CreateNetworkRequest) error {
	return driver.calNetDriver.CreateNetwork(request)
}

// DeleteNetwork .
func (driver *networkDriver) DeleteNetwork(request *network.DeleteNetworkRequest) error {
	return driver.calNetDriver.DeleteNetwork(request)
}

// CreateEndpoint .
func (driver *networkDriver) CreateEndpoint(request *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	return driver.calNetDriver.CreateEndpoint(request)
}

// DeleteEndpoint .
func (driver *networkDriver) DeleteEndpoint(request *network.DeleteEndpointRequest) error {
	return driver.calNetDriver.DeleteEndpoint(request)
}

// EndpointInfo .
func (driver *networkDriver) EndpointInfo(request *network.InfoRequest) (*network.InfoResponse, error) {
	return driver.calNetDriver.EndpointInfo(request)
}

// Join .
func (driver *networkDriver) Join(request *network.JoinRequest) (*network.JoinResponse, error) {
	return driver.calNetDriver.Join(request)
}

// Leave .
func (driver *networkDriver) Leave(request *network.LeaveRequest) error {
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

func (driver *networkDriver) findDockerContainerByEndpointID(endpointID string) (dockerTypes.Container, *dockerNetworkTypes.EndpointSettings, error) {
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
func (driver *networkDriver) DiscoverNew(request *network.DiscoveryNotification) error {
	return driver.calNetDriver.DiscoverNew(request)
}

// DiscoverDelete .
func (driver *networkDriver) DiscoverDelete(request *network.DiscoveryNotification) error {
	return driver.calNetDriver.DiscoverDelete(request)
}

// ProgramExternalConnectivity .
func (driver *networkDriver) ProgramExternalConnectivity(request *network.ProgramExternalConnectivityRequest) error {
	return driver.calNetDriver.ProgramExternalConnectivity(request)
}

// RevokeExternalConnectivity .
func (driver *networkDriver) RevokeExternalConnectivity(request *network.RevokeExternalConnectivityRequest) error {
	return driver.calNetDriver.RevokeExternalConnectivity(request)
}
