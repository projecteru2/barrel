package vessel

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/vessel/codecs"
)

// Helper .
type Helper struct {
	Vessel
	store.Store
}

// NewHelper .
func NewHelper(vess Vessel, stor store.Store) Helper {
	return Helper{
		Vessel: vess,
		Store:  stor,
	}
}

// ReleaseContainerAddressesByIPPools .
func (helper Helper) ReleaseContainerAddressesByIPPools(ctx context.Context, containerID string, pools []types.Pool) error {
	logger := helper.logger("ReleaseContainerAddressesByIPPools")
	logger.Infof("Release reserved IP by tied containerID(%s)", containerID)

	container := types.ContainerInfo{Container: types.Container{ID: containerID, HostName: helper.Hostname()}}
	codec := codecs.ContainerInfoCodec{Info: &container}
	var (
		addresses []types.IP
		releases  []types.IP
		updated   bool
		err       error
	)
	for !updated {
		addresses = nil
		releases = nil
		if err := helper.Get(ctx, &codec); err != nil {
			if store.IsNotExists(err) {
				logger.Infof("the container(%s) is not exists, will do nothing", containerID)
				return nil
			}
			return err
		}
		if len(container.Addresses) == 0 {
			logger.Infof("the ip of container(%s) is empty, will do nothing", containerID)
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
		if updated, err = helper.UpdateElseGet(ctx, &codec); err != nil {
			return err
		}
	}
	for _, address := range releases {
		if err := helper.FixedIPAllocator().UnallocFixedIP(ctx, address); err != nil {
			logger.Errorf("release reserved address error, cause = %v", err)
		}
	}
	return nil
}

// ReleaseContainerAddresses .
func (helper Helper) ReleaseContainerAddresses(ctx context.Context, containerID string) error {
	logger := helper.logger("ReleaseContainerAddresses")

	logger.Infof("Release reserved IP by tied containerID(%s)", containerID)
	container := types.Container{ID: containerID, HostName: helper.Hostname()}
	info := types.ContainerInfo{Container: container}
	if err := helper.GetAndDelete(ctx, &codecs.ContainerInfoCodec{Info: &info}); err != nil {
		if store.IsNotExists(err) {
			logger.Infof("the container(%s) is not exists, will do nothing\n", containerID)
			return nil
		}
		return err
	}
	if len(info.Addresses) == 0 {
		logger.Infof("the ip of container(%s) is empty, will do nothing\n", containerID)
		return nil
	}
	for _, address := range info.Addresses {
		if err := helper.FixedIPAllocator().ReturnFixedIP(ctx, address, container); err != nil {
			logger.WithError(err).WithField("fixed-ip", address).WithField("container", container).Error("Return fixed ip error")
		}
		if err := helper.FixedIPAllocator().UnallocFixedIP(ctx, address); err != nil {
			logger.Errorf("release reserved address error, cause = %v", err)
		}
	}
	return nil
}

// ReserveAddressForContainer .
func (helper Helper) ReserveAddressForContainer(ctx context.Context, containerID string, address types.IP) error {
	logger := helper.logger("ReserveAddressForContainer")

	if err := helper.FixedIPAllocator().AllocFixedIP(ctx, address); err != nil {
		logger.Errorf("Alocate fixed address(%v) error", address)
		return err
	}

	for {
		var (
			err           error
			containerInfo = types.ContainerInfo{Container: types.Container{ID: containerID, HostName: helper.Hostname()}}
			codec         = codecs.ContainerInfoCodec{Info: &containerInfo}
		)
		if err = helper.Get(ctx, &codec); store.ErrButOtherThenKVUnexistsErr(err) {
			return err
		}
		if err != nil {
			containerInfo.Addresses = []types.IP{address}
		} else {
			for _, addr := range containerInfo.Addresses {
				if addr.PoolID == address.PoolID && addr.Address == address.Address {
					return nil
				}
			}
			containerInfo.Addresses = append(containerInfo.Addresses, address)
		}
		if succeed, err := helper.UpdateElseGet(ctx, &codec); err != nil {
			return err
		} else if succeed {
			return nil
		}
	}
}

// InitContainerInfoRecord .
func (helper Helper) InitContainerInfoRecord(ctx context.Context, container types.Container, fixedIPs []types.IP) error {
	containerInfo := types.ContainerInfo{Container: container, Addresses: fixedIPs}
	for _, ip := range fixedIPs {
		if err := helper.FixedIPAllocator().BorrowFixedIP(ctx, ip, container); err != nil {
			log.WithError(err).WithField("FixedIP", ip).WithField("Container", container).Error("Borrow fixedip")
		}
	}
	return helper.Put(ctx, &codecs.ContainerInfoCodec{Info: &containerInfo})
}

func (helper Helper) logger(method string) *log.Entry {
	return log.WithField("Receiver", "VesselHelper").WithField("Method", method)
}
