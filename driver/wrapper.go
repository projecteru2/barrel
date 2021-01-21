package driver

import (
	pluginIPAM "github.com/docker/go-plugins-helpers/ipam"
	"github.com/docker/go-plugins-helpers/network"
	pluginNetwork "github.com/docker/go-plugins-helpers/network"
	logutils "github.com/projectcalico/libnetwork-plugin/utils/log"

	log "github.com/sirupsen/logrus"
)

type ipamWrapper struct {
	ipam pluginIPAM.Ipam
}

// GetCapabilities .
func (wrapper ipamWrapper) GetCapabilities() (*pluginIPAM.CapabilitiesResponse, error) {
	log.Info("GetCapabilities")
	resp := &pluginIPAM.CapabilitiesResponse{}
	logutils.JSONMessage("GetCapabilities response", resp)
	return resp, nil
}

// GetDefaultAddressSpaces .
func (wrapper ipamWrapper) GetDefaultAddressSpaces() (resp *pluginIPAM.AddressSpacesResponse, err error) {
	log.Info("GetDefaultAddressSpaces")
	if resp, err = wrapper.ipam.GetDefaultAddressSpaces(); err != nil {
		log.Errorf("GetDefaultAddressSpaces error, cause=%v", err)
		return
	}
	logutils.JSONMessage("GetDefaultAddressSpace response", resp)
	return
}

// RequestPool .
func (wrapper ipamWrapper) RequestPool(request *pluginIPAM.RequestPoolRequest) (resp *pluginIPAM.RequestPoolResponse, err error) {
	logutils.JSONMessage("RequestPool", request)
	if resp, err = wrapper.ipam.RequestPool(request); err != nil {
		log.Errorf("RequestPool error, cause=%v", err)
		return
	}
	logutils.JSONMessage("RequestPool response", resp)
	return
}

// ReleasePool .
func (wrapper ipamWrapper) ReleasePool(request *pluginIPAM.ReleasePoolRequest) error {
	logutils.JSONMessage("ReleasePool", request)
	return nil
}

// RequestAddress .
func (wrapper ipamWrapper) RequestAddress(request *pluginIPAM.RequestAddressRequest) (resp *pluginIPAM.RequestAddressResponse, err error) {
	logutils.JSONMessage("RequestAddress", request)
	if resp, err = wrapper.ipam.RequestAddress(request); err != nil {
		log.Errorf("RequestAddress error, cause=%v", err)
		return
	}
	logutils.JSONMessage("RequestAddress response", resp)
	return
}

// ReleaseAddress .
func (wrapper ipamWrapper) ReleaseAddress(request *pluginIPAM.ReleaseAddressRequest) error {
	logutils.JSONMessage("ReleaseAddress", request)
	err := wrapper.ipam.ReleaseAddress(request)
	if err == nil {
		log.Info("ReleaseAddress success")
	}
	return err
}

type driverWrapper struct {
	driver pluginNetwork.Driver
}

// GetCapabilities .
func (wrapper driverWrapper) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Info("GetCapabilities")
	resp := &network.CapabilitiesResponse{Scope: "global"}
	logutils.JSONMessage("GetCapabilities response", resp)
	return resp, nil
}

// AllocateNetwork is used for swarm-mode support in remote plugins, which
// Calico's libnetwork-plugin doesn't currently support.
func (wrapper driverWrapper) AllocateNetwork(request *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	logutils.JSONMessage("AllocateNetwork", request)
	var resp network.AllocateNetworkResponse
	logutils.JSONMessage("AllocateNetwork response", resp)
	return &resp, nil
}

// FreeNetwork is used for swarm-mode support in remote plugins, which
// Calico's libnetwork-plugin doesn't currently support.
func (wrapper driverWrapper) FreeNetwork(request *network.FreeNetworkRequest) error {
	logutils.JSONMessage("FreeNetwork request", request)
	return nil
}

// CreateNetwork .
func (wrapper driverWrapper) CreateNetwork(request *network.CreateNetworkRequest) error {
	logutils.JSONMessage("CreateNetwork", request)
	err := wrapper.driver.CreateNetwork(request)
	if err == nil {
		log.Info("CreateNetwork success")
	}
	return err
}

// DeleteNetwork .
func (wrapper driverWrapper) DeleteNetwork(request *network.DeleteNetworkRequest) error {
	logutils.JSONMessage("DeleteNetwork", request)
	return nil
}

// CreateEndpoint .
func (wrapper driverWrapper) CreateEndpoint(request *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	logutils.JSONMessage("CreateEndpoint", request)
	resp, err := wrapper.driver.CreateEndpoint(request)
	if err == nil {
		logutils.JSONMessage("CreateEndpoint response", resp)
	}
	return resp, err
}

// DeleteEndpoint .
func (wrapper driverWrapper) DeleteEndpoint(request *network.DeleteEndpointRequest) error {
	logutils.JSONMessage("DeleteEndpoint", request)
	err := wrapper.driver.DeleteEndpoint(request)
	if err == nil {
		log.Info("DeleteEndpoint success")
	}
	return err
}

// EndpointInfo .
func (wrapper driverWrapper) EndpointInfo(request *network.InfoRequest) (*network.InfoResponse, error) {
	logutils.JSONMessage("EndpointInfo", request)
	return nil, nil
}

// Join .
func (wrapper driverWrapper) Join(request *network.JoinRequest) (*network.JoinResponse, error) {
	logutils.JSONMessage("Join", request)
	resp, err := wrapper.driver.Join(request)
	if err == nil {
		logutils.JSONMessage("Join response", resp)
	}
	return resp, err
}

// Leave .
func (wrapper driverWrapper) Leave(request *network.LeaveRequest) error {
	logutils.JSONMessage("Leave", request)
	err := wrapper.driver.Leave(request)
	if err == nil {
		log.Info("Leave success")
	}
	return err
}

// DiscoverNew .
func (wrapper driverWrapper) DiscoverNew(request *network.DiscoveryNotification) error {
	logutils.JSONMessage("DiscoverNew", request)
	return nil
}

// DiscoverDelete .
func (wrapper driverWrapper) DiscoverDelete(request *network.DiscoveryNotification) error {
	logutils.JSONMessage("DiscoverDelete", request)
	err := wrapper.driver.DiscoverDelete(request)
	if err == nil {
		log.Info("DiscoverDelete success")
	}
	return err
}

// ProgramExternalConnectivity .
func (wrapper driverWrapper) ProgramExternalConnectivity(*network.ProgramExternalConnectivityRequest) error {
	return nil
}

// RevokeExternalConnectivity .
func (wrapper driverWrapper) RevokeExternalConnectivity(*network.RevokeExternalConnectivityRequest) error {
	return nil
}
