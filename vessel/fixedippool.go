package vessel

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel/codecs"
)

const (
	retryMaxCount = 3
)

// FixedIPPoolManager .
type FixedIPPoolManager interface {
	BorrowFixedIP(context.Context, types.IP, types.Container) error
	ReturnFixedIP(context.Context, types.IP, types.Container) error
	// Only assign when fixed ip is allocated
	AssignFixedIP(context.Context, types.IP) error
	UnassignFixedIP(context.Context, types.IP) error
	UnallocFixedIP(context.Context, types.IP, bool) error
	GetFixedIP(context.Context, types.IP, func(context.Context, types.IP, *codecs.IPInfoCodec) error) (*codecs.IPInfoCodec, error)
}

// FixedIPPool .
type FixedIPPool interface {
	CalicoIPPool
	FixedIPPoolManager
}

// FixedIPAllocator .
type FixedIPAllocator interface {
	CalicoIPAllocator
	FixedIPPoolManager
	AllocFixedIP(context.Context, types.IP) error
	AllocFixedIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error)
}

type fixedIPPool struct {
	CalicoIPPool
	store.Store
}

// NewFixedIPPool .
func NewFixedIPPool(pool CalicoIPPool, stor store.Store) FixedIPPool {
	return fixedIPPool{
		CalicoIPPool: pool,
		Store:        stor,
	}
}

// AssignFixedIP .
func (pool fixedIPPool) AssignFixedIP(ctx context.Context, ip types.IP) error {
	logger := pool.logger("AssignFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		ok          bool
		err         error
	)

	if ipInfoCodec, err = pool.GetFixedIP(ctx, ip, nil); err != nil {
		return err
	}
	if ipInfoCodec.IPInfo.Status.Match(types.IPStatusInUse) {
		logger.WithField("fixed-ip", ip).Error(`Fixed-ip in use`)
		return types.ErrIPInUse
	}

	ipInfoCodec.IPInfo.Status.Mark(types.IPStatusInUse)
	if ok, err = pool.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Error("Update IPInfo error")
		return err
	} else if !ok {
		// update failed, the ip is modified by another container
		return types.ErrIPInUse
	}
	return nil
}

// UnassignFixedIP .
func (pool fixedIPPool) UnassignFixedIP(ctx context.Context, ip types.IP) error {
	logger := pool.logger(
		"UnassignFixedIP",
	).WithField(
		"PoolID", ip.PoolID,
	).WithField(
		"Address", ip.Address,
	)
	ctx = utils.WithEntry(ctx, logger)

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		ok          bool
		err         error
	)

	if ipInfoCodec, err = pool.GetFixedIP(ctx, ip, nil); err != nil {
		return err
	}

	if !ipInfoCodec.IPInfo.Status.Match(types.IPStatusInUse) {
		logger.Warn("FixedIP is already unassigned")
		return nil
	}

	ipInfoCodec.IPInfo.Status.Unmark(types.IPStatusInUse)
	if ok, err = pool.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Errorf("Update IPInfo error")
		return err
	} else if !ok {
		return types.ErrIPInUse
	}

	return nil
}

// BorrowFixedIP .
func (pool fixedIPPool) BorrowFixedIP(ctx context.Context, ip types.IP, container types.Container) error {
	logger := pool.logger("BorrowFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	logger.WithField("ip", ip).WithField("container", container).Info("BorrowFixedIP")

	cnt := 0
	for cnt < retryMaxCount {
		codec, err := pool.GetFixedIP(ctx, ip, nil)
		if err != nil {
			return err
		}
		attrs := codec.IPInfo.Attrs
		if attrs == nil {
			attrs = &types.IPAttributes{}
			codec.IPInfo.Attrs = attrs
		}
		attrs.Borrowers = append(attrs.Borrowers, container)
		if ok, err := pool.UpdateElseGet(ctx, codec); err != nil {
			return err
		} else if ok {
			return nil
		}
		cnt++
	}
	return types.ErrMaxRetryCountExceeded
}

// ReturnFixedIP .
func (pool fixedIPPool) ReturnFixedIP(ctx context.Context, ip types.IP, container types.Container) error {
	logger := pool.logger("ReturnFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	cnt := 0
	for cnt < retryMaxCount {
		codec, err := pool.GetFixedIP(ctx, ip, nil)
		if err != nil {
			return err
		}
		attrs := codec.IPInfo.Attrs
		if attrs == nil || attrs.Borrowers == nil || len(attrs.Borrowers) == 0 {
			logger.WithField("FixedIP", ip).WithField("Container", container).Warn("Fixed ip already returned")
			return nil
		}
		var borrowers []types.Container
		for _, borrower := range attrs.Borrowers {
			if borrower != container {
				borrowers = append(borrowers, borrower)
			}
		}
		attrs.Borrowers = borrowers
		if ok, err := pool.UpdateElseGet(ctx, codec); err != nil {
			return err
		} else if ok {
			return nil
		}
		cnt++
	}
	return types.ErrMaxRetryCountExceeded
}

// UnallocFixedIP .
func (pool fixedIPPool) UnallocFixedIP(ctx context.Context, ip types.IP, force bool) error {
	logger := pool.logger(
		"UnallocFixedIP",
	).WithField(
		"PoolID", ip.PoolID,
	).WithField(
		"Address", ip.Address,
	)

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		ok          bool
		err         error
	)
	// if ok, err = alloc.store.Get(ctx, ipInfoCodec); err != nil {
	// 	logger.Errorf("Get IPInfo error, cause=%v", err)
	// 	return err
	// }

	// if !ok {
	// 	// The fixed ip is not allocated, so give a warning here
	// 	logger.Warnf(`IP is not allocated, {"PoolID": "%s", "Address": "%s"}`, ip.PoolID, ip.Address)
	// 	return types.ErrFixedIPNotAllocated
	// }

	if ipInfoCodec, err = pool.GetFixedIP(ctx, ip, nil); err != nil {
		return err
	}

	ipInfo := ipInfoCodec.IPInfo
	if ipInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	}

	// if force remove, we will not check borrowers
	if !force && ipInfo.Attrs != nil && len(ipInfo.Attrs.Borrowers) > 0 {
		logger.Info("IP still has borrower")
		return types.ErrFixedIPHasBorrower
	}

	// Lock the ip first
	ipInfo.Status.Mark(types.IPStatusInUse, types.IPStatusRetired)
	if ok, err = pool.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Errorf("Lock IPInfo failed")
	} else if !ok {
		return types.ErrIPInUse
	}

	// Now we remove the ipInfo
	if err = pool.Delete(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Error("Delete IPInfo failed")
		// The
		return err
	}

	// Now we free the address
	if err = pool.UnallocIP(ctx, ip); err != nil {
		logger.WithError(err).Error("Unalloc IP failed")
		return err
	}

	return nil
}

func (pool fixedIPPool) logger(method string) *log.Entry {
	return log.WithField("Receiver", "fixedIPPool").WithField("Method", method)
}

func (pool fixedIPPool) GetFixedIP(
	ctx context.Context,
	ip types.IP,
	allocateFixedIP func(context.Context, types.IP, *codecs.IPInfoCodec) error,
) (*codecs.IPInfoCodec, error) {
	logger := utils.LogEntry(ctx)
	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codecs.IPInfoCodec{IPInfo: &ipInfo}
	)
	// nolint:nestif
	if err := pool.Get(ctx, ipInfoCodec); err != nil {
		if store.IsNotExists(err) {
			if allocateFixedIP == nil {
				logger.Error("IP is not allocated")
				return nil, types.ErrFixedIPNotAllocated
			}
			if err := allocateFixedIP(ctx, ip, ipInfoCodec); err != nil {
				return nil, err
			}
			return ipInfoCodec, nil
		}
		logger.WithError(err).Error("Get fixed-ip info error")
		return nil, err
	}
	return ipInfoCodec, nil
}

// func (pool fixedIPPool) context(ctx context.Context, method string) context.Context {
// 	return utils.WithEntry(ctx, pool.logger(method))
// }

type fixedIPAllocator struct {
	fixedIPPool
	CalicoIPAllocator
}

// NewFixedIPAllocator .
func NewFixedIPAllocator(allocator CalicoIPAllocator, stor store.Store) FixedIPAllocator {
	return fixedIPAllocator{
		CalicoIPAllocator: allocator,
		fixedIPPool: fixedIPPool{
			CalicoIPPool: allocator,
			Store:        stor,
		},
	}
}

// AllocFixedIP .
func (alloc fixedIPAllocator) AllocFixedIP(ctx context.Context, ip types.IP) error {
	ctx = alloc.context(ctx, "AllocFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		err         error
	)
	if ipInfoCodec, err = alloc.GetFixedIP(ctx, ip, alloc.createFixedIP); err != nil {
		return err
	}

	// ipInfoCodec will not be nil when we call getFixedIP by alloIfAbsent=true parameter
	// still we run a check not to make programe panic
	if ipInfoCodec == nil {
		return types.ErrCriticalError
	}

	if ipInfoCodec.IPInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	}
	return nil
}

// AllocFixedIPFromPools .
func (alloc fixedIPAllocator) AllocFixedIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error) {
	logger := alloc.logger("AllocFixedIPFromPools")
	var (
		ip  types.IPAddress
		err error
	)
	if ip, err = alloc.AllocIPFromPools(ctx, pools); err != nil {
		return ip, err
	}
	var (
		ipInfo      = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec = &codecs.IPInfoCodec{IPInfo: &ipInfo}
	)
	if err = alloc.Put(ctx, ipInfoCodec); err != nil {
		if err := alloc.UnallocIP(ctx, ip.IP); err != nil {
			logger.WithError(err).Errorf("UnallocIP error")
		}
		return ip, err
	}
	return ip, nil
}

func (alloc fixedIPAllocator) createFixedIP(ctx context.Context, ip types.IP, codec *codecs.IPInfoCodec) error {
	logger := utils.LogEntry(ctx)
	if err := alloc.AllocIP(ctx, ip); err != nil {
		logger.WithError(err).Error("Alloc IP error")
		return err
	}
	if err := alloc.Put(ctx, codec); err != nil {
		logger.WithError(err).Error("Create FixedIPInfo error")
		return err
	}
	return nil
}

func (alloc fixedIPAllocator) logger(method string) *log.Entry {
	return log.WithField("Receiver", "fixedIPAllocator").WithField("Method", method)
}

func (alloc fixedIPAllocator) context(ctx context.Context, method string) context.Context {
	return utils.WithEntry(ctx, alloc.logger(method))
}
