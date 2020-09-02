package ipam

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projecteru2/barrel/common"
	barrelEtcd "github.com/projecteru2/minions/barrel/etcd"
	calicoIPAM "github.com/projecteru2/minions/driver/calico/ipam"
	minionsTypes "github.com/projecteru2/minions/types"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrUnsupervisedNetwork .
	ErrUnsupervisedNetwork = errors.New("unsupervised network")
	// ErrConfiguredPoolUnfound .
	ErrConfiguredPoolUnfound = errors.New("network doesn't contains configured ip pools")
)

// IPAM .
type IPAM struct {
	Driver           string
	CalicoIPAMDriver calicoIPAM.CalicoIPAM
	BarrelEtcd       *barrelEtcd.Etcd
	DockerClient     *dockerClient.Client
}

// ReserveAddressFromPools .
func (ipam *IPAM) ReserveAddressFromPools(pools []*minionsTypes.Pool) (common.ReservedAddress, error) {
	var (
		ip  caliconet.IP
		err error
	)
	if len(pools) == 1 {
		if ip, err = ipam.CalicoIPAMDriver.AutoAssign(pools[0].Name); err != nil {
			return common.ReservedAddress{}, err
		}
		address := minionsTypes.ReservedAddress{PoolID: pools[0].Name, Address: fmt.Sprintf("%v", ip)}
		return common.ReservedAddress{
			Version: ip.Version(),
			Address: address,
		}, ipam.ReserveAddress(address)
	}
	var poolNames []string
	for _, pool := range pools {
		if ip, err = ipam.CalicoIPAMDriver.AutoAssign(pool.Name); err != nil {
			poolNames = append(poolNames, pool.Name)
			log.Errorf("[IPAM::ReserveAddressFromPools] AutoAssign from %s error, %v", pool.Name, err)
			continue
		}
		address := minionsTypes.ReservedAddress{PoolID: pool.Name, Address: fmt.Sprintf("%v", ip)}
		return common.ReservedAddress{
			Version: ip.Version(),
			Address: address,
		}, ipam.ReserveAddress(address)
	}
	return common.ReservedAddress{}, errors.Errorf("[IPAM::ReserveAddressFromPools] AutoAssign from %v failed", poolNames)
}

// ReserveAddress .
func (ipam *IPAM) ReserveAddress(address minionsTypes.ReservedAddress) error {
	if err := ipam.BarrelEtcd.Put(
		context.Background(),
		&barrelEtcd.ReservedAddressCodec{Address: &address},
	); err != nil {
		// don't try rollback here, 'cause put might be succeed
		// identify the error first(do it later)
		log.Errorf("[IPAM::ReserveAddress] reserve address(%v) error, %v", address, err)
		return err
	}
	return nil
}

// InitContainerInfoRecord .
func (ipam *IPAM) InitContainerInfoRecord(containerInfo minionsTypes.ContainerInfo) error {
	return ipam.BarrelEtcd.Put(context.Background(), &barrelEtcd.ContainerInfoCodec{Info: &containerInfo})
}

// AddReservedAddressForContainer .
func (ipam *IPAM) AddReservedAddressForContainer(containerID string, address minionsTypes.ReservedAddress) error {
	containerInfo := minionsTypes.ContainerInfo{ID: containerID}
	codec := barrelEtcd.ContainerInfoCodec{Info: &containerInfo}
	for {
		var (
			presented bool
			err       error
		)
		if presented, err = ipam.BarrelEtcd.Get(context.Background(), &codec); err != nil {
			return err
		}
		if !presented {
			containerInfo.Addresses = []minionsTypes.ReservedAddress{address}
		} else {
			containerInfo.Addresses = append(containerInfo.Addresses, address)
		}
		if succeed, err := ipam.BarrelEtcd.Update(context.Background(), &codec); err != nil {
			return err
		} else if succeed {
			return nil
		}
	}
}

// ReleaseContainer .
func (ipam *IPAM) ReleaseContainer(containerID string) error {
	log.Infof("Release reserved IP by tied containerID(%s)\n", containerID)

	container := minionsTypes.ContainerInfo{ID: containerID}
	if present, err := ipam.BarrelEtcd.GetAndDelete(context.Background(), &barrelEtcd.ContainerInfoCodec{Info: &container}); err != nil {
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
		ipam.ReleaseReservedAddress(address)
	}
	return nil
}

// ReleaseContainerByIPPools .
func (ipam *IPAM) ReleaseContainerByIPPools(containerID string, pools []*minionsTypes.Pool) error {
	log.Infof("Release reserved IP by tied containerID(%s)\n", containerID)

	container := minionsTypes.ContainerInfo{ID: containerID}
	codec := barrelEtcd.ContainerInfoCodec{Info: &container}
	var (
		addresses []minionsTypes.ReservedAddress
		releases  []minionsTypes.ReservedAddress
		updated   bool
		err       error
	)
	for !updated {
		addresses = nil
		releases = nil
		if present, err := ipam.BarrelEtcd.Get(context.Background(), &codec); err != nil {
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
		if updated, err = ipam.BarrelEtcd.Update(context.Background(), &codec); err != nil {
			return err
		}
	}
	for _, address := range releases {
		ipam.ReleaseReservedAddress(address)
	}
	return nil
}

func included(pools []*minionsTypes.Pool, poolID string) bool {
	for _, pool := range pools {
		if pool.Name == poolID {
			return true
		}
	}
	return false
}

// ReleaseReservedAddress .
func (ipam *IPAM) ReleaseReservedAddress(address minionsTypes.ReservedAddress) {
	log.Infof("acquiring reserved address(%s)\n", address.Address)
	if present, err := ipam.BarrelEtcd.GetAndDelete(context.Background(), &barrelEtcd.ReservedAddressCodec{Address: &address}); err != nil {
		log.Errorf("acquiring reserved address(%s) error, %v", address.Address, err)
	} else if !present {
		log.Infof("reserved address(%s) has already been released or reallocated", address.Address)
	}

	log.Infof("release ip(%s) to calico pools\n", address.Address)
	if err := ipam.CalicoIPAMDriver.ReleaseIP(address.PoolID, address.Address); err != nil {
		log.Errorf("Releasing address(%v) error, %v", address, err)
	}
}

// GetIPPoolsByNetworkName .
func (ipam *IPAM) GetIPPoolsByNetworkName(name string) ([]*minionsTypes.Pool, error) {
	var (
		network types.NetworkResource
		err     error
	)
	if network, err = ipam.DockerClient.NetworkInspect(context.Background(), name, dockerTypes.NetworkInspectOptions{}); err != nil {
		return nil, err
	}
	if network.Driver != ipam.Driver {
		return nil, ErrUnsupervisedNetwork
	}
	if len(network.IPAM.Config) == 0 {
		return nil, ErrConfiguredPoolUnfound
	}
	var cidrs []string
	for _, config := range network.IPAM.Config {
		cidrs = append(cidrs, config.Subnet)
	}
	return ipam.CalicoIPAMDriver.RequestPools(cidrs)
}
