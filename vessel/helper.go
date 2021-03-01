package vessel

import (
	"context"

	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel/codec"
)

// Helper .
type Helper struct {
	Vessel
	store.Store
	utils.LoggerFactory
}

// NewHelper .
func NewHelper(vess Vessel, stor store.Store) Helper {
	return Helper{
		Vessel:        vess,
		Store:         stor,
		LoggerFactory: utils.NewObjectLogger("VesselHelper"),
	}
}

// ReleaseContainerAddressesByIPPools .
func (helper Helper) ReleaseContainerAddressesByIPPools(ctx context.Context, containerID string, pools []types.Pool) error {
	logger := helper.Logger("ReleaseContainerAddressesByIPPools")
	logger.Infof("Release reserved IP by tied containerID(%s)", containerID)

	container := types.ContainerInfo{ID: containerID}
	codec := codec.ContainerInfoCodec{Info: &container}
	var (
		addresses []types.IP
		releases  []types.IP
		updated   bool
		err       error
	)
	for !updated {
		addresses = nil
		releases = nil
		if present, err := helper.Get(ctx, &codec); err != nil {
			return err
		} else if !present {
			logger.Infof("the container(%s) is not exists, will do nothing", containerID)
			return nil
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
		if updated, err = helper.Update(ctx, &codec); err != nil {
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
	logger := helper.Logger("ReleaseContainerAddresses")

	logger.Infof("Release reserved IP by tied containerID(%s)", containerID)

	container := types.ContainerInfo{ID: containerID, HostName: helper.Hostname()}
	if present, err := helper.GetAndDelete(context.Background(), &codec.ContainerInfoCodec{Info: &container}); err != nil {
		return err
	} else if !present {
		logger.Infof("the container(%s) is not exists, will do nothing\n", containerID)
		return nil
	}
	if len(container.Addresses) == 0 {
		logger.Infof("the ip of container(%s) is empty, will do nothing\n", containerID)
		return nil
	}
	for _, address := range container.Addresses {
		if err := helper.FixedIPAllocator().UnallocFixedIP(ctx, address); err != nil {
			logger.Errorf("release reserved address error, cause = %v", err)
		}
	}
	return nil
}

// ReserveAddressForContainer .
func (helper Helper) ReserveAddressForContainer(ctx context.Context, containerID string, address types.IP) error {
	logger := helper.Logger("ReserveAddressForContainer")

	if err := helper.FixedIPAllocator().AllocFixedIP(ctx, address); err != nil {
		logger.Errorf("Alocate fixed address(%v) error", address)
		return err
	}

	for {
		var (
			presented     bool
			err           error
			containerInfo = types.ContainerInfo{ID: containerID}
			codec         = codec.ContainerInfoCodec{Info: &containerInfo}
		)
		if presented, err = helper.Get(ctx, &codec); err != nil {
			return err
		}
		if !presented {
			containerInfo.Addresses = []types.IP{address}
		} else {
			for _, addr := range containerInfo.Addresses {
				if addr.PoolID == address.PoolID && addr.Address == address.Address {
					return nil
				}
			}
			containerInfo.Addresses = append(containerInfo.Addresses, address)
		}
		if succeed, err := helper.Update(ctx, &codec); err != nil {
			return err
		} else if succeed {
			return nil
		}
	}
}

// InitContainerInfoRecord .
func (helper Helper) InitContainerInfoRecord(ctx context.Context, containerInfo types.ContainerInfo) error {
	return helper.Put(ctx, &codec.ContainerInfoCodec{Info: &containerInfo})
}
