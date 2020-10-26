package vessel

import (
	"context"

	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel/codec"
)

// FixedIPAllocator .
type FixedIPAllocator interface {
	CalicoIPAllocator
	AllocFixedIP(context.Context, types.IP) error
	UnallocFixedIP(context.Context, types.IP) error
	// Only assign when fixed ip is allocated
	AssignFixedIP(context.Context, types.IP) error
	UnassignFixedIP(context.Context, types.IP) error
	AllocFixedIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error)
}

// FixedIPAllocatorImpl .
type FixedIPAllocatorImpl struct {
	CalicoIPAllocator
	utils.LoggerFactory
	store.Store
}

// NewFixedIPAllocator .
func NewFixedIPAllocator(allocator CalicoIPAllocator, stor store.Store) FixedIPAllocator {
	return FixedIPAllocatorImpl{
		CalicoIPAllocator: allocator,
		LoggerFactory:     utils.NewObjectLogger("FixedIPAllocatorImpl"),
		Store:             stor,
	}
}

// AssignFixedIP .
func (impl FixedIPAllocatorImpl) AssignFixedIP(ctx context.Context, ip types.IP) error {
	logger := impl.Logger("AssignFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codec.IPInfoCodec{IPInfo: &ipInfo}
		ok          bool
		err         error
	)

	if ok, err = impl.Get(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Get IPInfo error, cause=%v", err)
		return err
	} else if !ok {
		logger.Warnf(`IP is not allocated, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
		return types.ErrFixedIPNotAllocated
	}
	if ipInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	}

	ipInfo.Status.Mark(types.IPStatusInUse)
	if ok, err = impl.Update(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Update IPInfo error, cause=%v", err)
		return err
	} else if !ok {
		return types.ErrIPInUse
	}

	return nil
}

// UnassignFixedIP .
func (impl FixedIPAllocatorImpl) UnassignFixedIP(ctx context.Context, ip types.IP) error {
	logger := impl.Logger("UnassignFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codec.IPInfoCodec{IPInfo: &ipInfo}
		ok          bool
		err         error
	)

	if ok, err = impl.Get(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Get IPInfo error, cause=%v", err)
		return err
	} else if !ok {
		logger.Warnf(`IP is not allocated, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
		return types.ErrFixedIPNotAllocated
	}

	if !ipInfo.Status.Match(types.IPStatusInUse) {
		logger.Warnf(`FixedIP is already unassigned, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
		return nil
	}

	ipInfo.Status.Mark(types.IPStatusInUse)
	if ok, err = impl.Update(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Update IPInfo error, cause=%v", err)
		return err
	} else if !ok {
		return types.ErrIPInUse
	}

	return nil
}

// AllocFixedIP .
func (impl FixedIPAllocatorImpl) AllocFixedIP(ctx context.Context, ip types.IP) error {
	logger := impl.Logger("AllocFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codec.IPInfoCodec{IPInfo: &ipInfo}
		ok          bool
		err         error
	)
	if ok, err = impl.Get(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Get IPInfo error, cause=%v", err)
		return err
	}
	if ok && ipInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	} else if ok {
		logger.Warnf(`FixedIP is already allocated, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
		return nil
	}

	if err = impl.AllocIP(ctx, ip); err != nil {
		logger.Errorf("Alloc IP error, cause=%v", err)
		return err
	}
	if err = impl.Put(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Create FixedIPInfo error, cause=%v", err)
		return err
	}
	return nil
}

// UnallocFixedIP .
func (impl FixedIPAllocatorImpl) UnallocFixedIP(ctx context.Context, ip types.IP) error {
	logger := impl.Logger("UnallocFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codec.IPInfoCodec{IPInfo: &ipInfo}
		ok          bool
		err         error
	)
	if ok, err = impl.Get(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Get IPInfo error, cause=%v", err)
		return err
	}

	if !ok {
		// The fixed ip is not allocated, so give a warning here
		logger.Warnf(`IP is not allocated, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
		return types.ErrFixedIPNotAllocated
	}

	if ipInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	}

	// Lock the ip first
	ipInfo.Status.Mark(types.IPStatusInUse, types.IPStatusRetired)
	if ok, err = impl.Update(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Lock IPInfo failed, cause=%v", err)
	} else if !ok {
		return types.ErrIPInUse
	}

	// Now we remove the ipInfo
	if _, err = impl.Delete(ctx, ipInfoCodec); err != nil {
		logger.Errorf("Delete IPInfo failed, cause=%v", err)
		// The
		return err
	}

	// Now we free the address
	if err = impl.UnallocIP(ctx, ip); err != nil {
		logger.Errorf("Unalloc IP failed, cause=%v", err)
		return err
	}

	return nil
}

// AllocFixedIPFromPools .
func (impl FixedIPAllocatorImpl) AllocFixedIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error) {
	logger := impl.Logger("AllocFixedIPFromPools")
	var (
		ip  types.IPAddress
		err error
	)
	if ip, err = impl.AllocIPFromPools(ctx, pools); err != nil {
		return ip, err
	}
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codec.IPInfoCodec{IPInfo: &ipInfo}
	)
	if err = impl.Put(ctx, ipInfoCodec); err != nil {
		if err := impl.UnallocIP(ctx, ip.IP); err != nil {
			logger.Errorf("UnallocIP error, cause=%v", err)
		}
		return ip, err
	}
	return ip, nil
}
