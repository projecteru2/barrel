package calicoplus

import (
	"context"
	"fmt"
	"net"

	pluginIPAM "github.com/docker/go-plugins-helpers/ipam"
	"github.com/juju/errors"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	logutils "github.com/projectcalico/libnetwork-plugin/utils/log"

	calicoDriver "github.com/projecteru2/barrel/driver/calicoplus/calico"
	"github.com/projecteru2/barrel/driver/calicoplus/codec"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"

	log "github.com/sirupsen/logrus"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
)

type ipamDriver struct {
	driverName   string
	calicoIPAM   *calicoDriver.IPAMDriver
	barrelEtcd   store.Store
	dockerClient *dockerClient.Client
}

// GetCapabilities .
func (i *ipamDriver) GetCapabilities() (*pluginIPAM.CapabilitiesResponse, error) {
	resp := pluginIPAM.CapabilitiesResponse{}
	logutils.JSONMessage("GetCapabilities response", resp)
	return &resp, nil
}

// GetDefaultAddressSpaces .
func (i *ipamDriver) GetDefaultAddressSpaces() (*pluginIPAM.AddressSpacesResponse, error) {
	resp := &pluginIPAM.AddressSpacesResponse{
		LocalDefaultAddressSpace:  calicoDriver.CalicoLocalAddressSpace,
		GlobalDefaultAddressSpace: calicoDriver.CalicoGlobalAddressSpace,
	}
	logutils.JSONMessage("GetDefaultAddressSpace response", resp)
	return resp, nil
}

// RequestPool .
func (i *ipamDriver) RequestPool(request *pluginIPAM.RequestPoolRequest) (*pluginIPAM.RequestPoolResponse, error) {
	logutils.JSONMessage("RequestPool", request)

	// Calico IPAM does not allow you to request a SubPool.
	if request.SubPool != "" {
		err := errors.New(
			"Calico IPAM does not support sub pool configuration " +
				"on 'docker create network'. Calico IP Pools " +
				"should be configured first and IP assignment is " +
				"from those pre-configured pools.",
		)
		log.Errorln(err)
		return nil, err
	}

	if len(request.Options) != 0 {
		err := errors.New("Arbitrary options are not supported")
		log.Errorln(err)
		return nil, err
	}

	var (
		pool *types.Pool
		err  error
	)

	// If a pool (subnet on the CLI) is specified, it must match one of the
	// preconfigured Calico pools.
	if request.Pool != "" {
		if pool, err = i.calicoIPAM.RequestPool(request.Pool); err != nil {
			log.Errorf("[IPAMDriver::RequestPool] request calico pool error, %v", err)
			return nil, err
		}
	} else {
		pool = i.calicoIPAM.RequestDefaultPool(request.V6)
	}

	// We use static pool ID and CIDR. We don't need to signal the
	// The meta data includes a dummy gateway address. This prevents libnetwork
	// from requesting a gateway address from the pool since for a Calico
	// network our gateway is set to a special IP.
	resp := &pluginIPAM.RequestPoolResponse{
		PoolID: pool.Name,
		Pool:   pool.CIDR,
		Data:   map[string]string{"com.docker.network.gateway": pool.Gateway},
	}
	logutils.JSONMessage("RequestPool response", resp)
	return resp, nil
}

// ReleasePool .
func (i *ipamDriver) ReleasePool(request *pluginIPAM.ReleasePoolRequest) error {
	logutils.JSONMessage("ReleasePool", request)
	return nil
}

// RequestAddress .
func (i *ipamDriver) RequestAddress(request *pluginIPAM.RequestAddressRequest) (*pluginIPAM.RequestAddressResponse, error) {
	logutils.JSONMessage("RequestAddress", request)

	// Calico IPAM does not allow you to choose a gateway.
	if err := checkOptions(request.Options); err != nil {
		log.Errorf("[IpamDriver::RequestAddress] check request options failed, %v", err)
		return nil, err
	}

	var address caliconet.IP
	var err error
	if address, err = i.requestIP(request); err != nil {
		return nil, err
	}

	resp := &pluginIPAM.RequestAddressResponse{
		// Return the IP as a CIDR.
		Address: formatIPAddress(address),
	}
	logutils.JSONMessage("RequestAddress response", resp)
	return resp, nil
}

// ReleaseAddress .
func (i *ipamDriver) ReleaseAddress(request *pluginIPAM.ReleaseAddressRequest) error {
	logutils.JSONMessage("ReleaseAddress", request)
	reserved, err := i.IsAddressReserved(
		&types.Address{
			PoolID:  request.PoolID,
			Address: request.Address,
		},
	)
	if err != nil {
		log.Errorf("Get reserved ip status error, ip: %v", request.Address)
		return err
	}

	if reserved {
		log.Infof("Ip is reserved, will not release to pool, ip: %v\n", request.Address)
		return nil
	}
	return i.calicoIPAM.ReleaseIP(request.PoolID, request.Address)
}

func (i *ipamDriver) requestIP(request *pluginIPAM.RequestAddressRequest) (caliconet.IP, error) {
	if request.Address == "" {
		return i.calicoIPAM.AutoAssign(request.PoolID)
	}
	var err error

	// specified address requested, so will try assign from reserved pool, then calico pool
	log.Info("Assigning specified IP from reserved pool first, then calico pools")

	// try to acquire ip from reserved ip pool
	var acquired bool
	if acquired, err = i.AquireIfReserved(
		&types.Address{
			PoolID:  request.PoolID,
			Address: request.Address,
		}); err != nil {
		return caliconet.IP{}, err
	}
	if acquired {
		return caliconet.IP{IP: net.ParseIP(request.Address)}, nil
	}
	// assign IP from calico
	return i.calicoIPAM.AssignIP(request.Address)
}

// ReserveAddressFromPools .
func (i *ipamDriver) ReserveAddressFromPools(pools []types.Pool) (types.AddressWithVersion, error) {
	var (
		ip  caliconet.IP
		err error
	)
	if len(pools) == 1 {
		if ip, err = i.calicoIPAM.AutoAssign(pools[0].Name); err != nil {
			return types.AddressWithVersion{}, err
		}
		address := types.Address{PoolID: pools[0].Name, Address: fmt.Sprintf("%v", ip)}
		return types.AddressWithVersion{
			Version: ip.Version(),
			Address: address,
		}, i.ReserveAddress(address)
	}
	var poolNames []string
	for _, pool := range pools {
		if ip, err = i.calicoIPAM.AutoAssign(pool.Name); err != nil {
			poolNames = append(poolNames, pool.Name)
			log.Errorf("[IPAM::ReserveAddressFromPools] AutoAssign from %s error, %v", pool.Name, err)
			continue
		}
		address := types.Address{PoolID: pool.Name, Address: fmt.Sprintf("%v", ip)}
		return types.AddressWithVersion{
			Version: ip.Version(),
			Address: address,
		}, i.ReserveAddress(address)
	}
	return types.AddressWithVersion{}, errors.Errorf("[IPAM::ReserveAddressFromPools] AutoAssign from %v failed", poolNames)
}

// ReserveAddress .
func (i *ipamDriver) ReserveAddress(address types.Address) error {
	if err := i.barrelEtcd.Put(
		context.Background(),
		&codec.ReservedAddressCodec{Address: &address},
	); err != nil {
		// don't try rollback here, 'cause put might be succeed
		// identify the error first(do it later)
		log.Errorf("[IPAM::ReserveAddress] reserve address(%v) error, %v", address, err)
		return err
	}
	return nil
}

// InitContainerInfoRecord .
func (i *ipamDriver) InitContainerInfoRecord(containerInfo types.ContainerInfo) error {
	return i.barrelEtcd.Put(context.Background(), &codec.ContainerInfoCodec{Info: &containerInfo})
}

// AddAddressForContainer .
func (i *ipamDriver) ReserveAddressForContainer(containerID string, address types.Address) error {
	if err := i.barrelEtcd.Put(context.Background(), &codec.ReservedAddressCodec{Address: &address}); err != nil {
		log.Errorf("[ipamDriver::ReserveAddressForContainer] Put reserved address(%v) error", address)
		return err
	}
	for {
		var (
			presented     bool
			err           error
			containerInfo = types.ContainerInfo{ID: containerID}
			codec         = codec.ContainerInfoCodec{Info: &containerInfo}
		)
		if presented, err = i.barrelEtcd.Get(context.Background(), &codec); err != nil {
			return err
		}
		if !presented {
			containerInfo.Addresses = []types.Address{address}
		} else {
			for _, addr := range containerInfo.Addresses {
				if addr.PoolID == address.PoolID && addr.Address == address.Address {
					return nil
				}
			}
			containerInfo.Addresses = append(containerInfo.Addresses, address)
		}
		if succeed, err := i.barrelEtcd.Update(context.Background(), &codec); err != nil {
			return err
		} else if succeed {
			return nil
		}
	}
}

// ReleaseContainer .
func (i *ipamDriver) ReleaseContainerAddresses(containerID string) error {
	log.Infof("Release reserved IP by tied containerID(%s)\n", containerID)

	container := types.ContainerInfo{ID: containerID}
	if present, err := i.barrelEtcd.GetAndDelete(context.Background(), &codec.ContainerInfoCodec{Info: &container}); err != nil {
		return err
	} else if !present {
		log.Infof("the container(%s) is not exists, will do nothing\n", containerID)
		return nil
	}
	if len(container.Addresses) == 0 {
		log.Infof("the ip of container(%s) is empty, will do nothing\n", containerID)
		return nil
	}
	for _, address := range container.Addresses {
		if err := i.ReleaseReservedAddress(address); err != nil {
			log.Errorf("[ipamDriver::ReleaseContainerAddresses] release reserved address error, cause = %v", err)
		}
	}
	return nil
}

// ReleaseContainerByIPPools .
func (i *ipamDriver) ReleaseContainerAddressesByIPPools(containerID string, pools []types.Pool) error {
	log.Infof("Release reserved IP by tied containerID(%s)\n", containerID)

	container := types.ContainerInfo{ID: containerID}
	codec := codec.ContainerInfoCodec{Info: &container}
	var (
		addresses []types.Address
		releases  []types.Address
		updated   bool
		err       error
	)
	for !updated {
		addresses = nil
		releases = nil
		if present, err := i.barrelEtcd.Get(context.Background(), &codec); err != nil {
			return err
		} else if !present {
			log.Infof("the container(%s) is not exists, will do nothing\n", containerID)
			return nil
		}
		if len(container.Addresses) == 0 {
			log.Infof("the ip of container(%s) is empty, will do nothing\n", containerID)
			return nil
		}
		for _, address := range container.Addresses {
			if !included(pools, address.PoolID) {
				addresses = append(addresses, address)
			} else {
				releases = append(releases, address)
			}
		}
		container.Addresses = addresses
		if updated, err = i.barrelEtcd.Update(context.Background(), &codec); err != nil {
			return err
		}
	}
	for _, address := range releases {
		if err := i.ReleaseReservedAddress(address); err != nil {
			log.Errorf("[ipamDriver::ReleaseContainerAddressesByIPPools] release reserved address error, cause = %v", err)
		}
	}
	return nil
}

func included(pools []types.Pool, poolID string) bool {
	for _, pool := range pools {
		if pool.Name == poolID {
			return true
		}
	}
	return false
}

// ReleaseReservedAddress .
func (i *ipamDriver) ReleaseReservedAddress(address types.Address) error {
	log.Infof("[ipamDriver::ReleaseReservedAddress] Releasing address(%s)", address.Address)
	if present, err := i.barrelEtcd.GetAndDelete(context.Background(), &codec.ReservedAddressCodec{Address: &address}); err != nil {
		log.Errorf("[ipamDriver::ReleaseReservedAddress] Acquiring reserved address(%s) error", address.Address)
		return err
	} else if !present {
		log.Infof("[ipamDriver::ReleaseReservedAddress] Reserved address(%s) has already been released or reallocated", address.Address)
		return nil
	}

	log.Infof("[ipamDriver::ReleaseReservedAddress] Release ip(%s) to calico pools", address.Address)
	if err := i.calicoIPAM.ReleaseIP(address.PoolID, address.Address); err != nil {
		log.Errorf("[ipamDriver::ReleaseReservedAddress] Releasing address(%v) error", address)
		return err
	}
	log.Infof("[ipamDriver::ReleaseReservedAddress] Release ip(%s) success", address.Address)
	return nil
}

// GetIPPoolsByNetworkName .
func (i *ipamDriver) GetIPPoolsByNetworkName(name string) ([]types.Pool, error) {
	var (
		network dockerTypes.NetworkResource
		err     error
	)
	if network, err = i.dockerClient.NetworkInspect(context.Background(), name, dockerTypes.NetworkInspectOptions{}); err != nil {
		return nil, err
	}
	if network.Driver != i.driverName {
		return nil, types.ErrUnsupervisedNetwork
	}
	if len(network.IPAM.Config) == 0 {
		return nil, types.ErrConfiguredPoolUnfound
	}
	var cidrs []string
	for _, config := range network.IPAM.Config {
		cidrs = append(cidrs, config.Subnet)
	}
	return i.calicoIPAM.RequestPools(cidrs)
}

// IPIsReserved .
func (i *ipamDriver) IsAddressReserved(address *types.Address) (bool, error) {
	return i.barrelEtcd.Get(context.Background(), &codec.ReservedAddressCodec{Address: address})
}

// AquireIfReserved .
func (i *ipamDriver) AquireIfReserved(address *types.Address) (bool, error) {
	return i.barrelEtcd.Delete(context.Background(), &codec.ReservedAddressCodec{Address: address})
}
