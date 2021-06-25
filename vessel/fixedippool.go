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

// FixedIPAllocator .
type FixedIPAllocator interface {
	CalicoIPAllocator
	AllocFixedIP(context.Context, types.IP) error
	UnallocFixedIP(context.Context, types.IP) error
	// Only assign when fixed ip is allocated
	AssignFixedIP(context.Context, types.IP) error
	UnassignFixedIP(context.Context, types.IP) error
	AllocFixedIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error)
	BorrowFixedIP(context.Context, types.IP, types.Container) error
	ReturnFixedIP(context.Context, types.IP, types.Container) error
}

// fixedIPAllocator .
type fixedIPAllocator struct {
	CalicoIPAllocator
	store store.Store
}

// NewFixedIPAllocator .
func NewFixedIPAllocator(allocator CalicoIPAllocator, stor store.Store) FixedIPAllocator {
	return fixedIPAllocator{
		CalicoIPAllocator: allocator,
		store:             stor,
	}
}

// AssignFixedIP .
func (alloc fixedIPAllocator) AssignFixedIP(ctx context.Context, ip types.IP) error {
	logger := alloc.logger("AssignFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		ok          bool
		err         error
	)

	if ipInfoCodec, err = alloc.getFixedIP(ctx, ip, false); err != nil {
		return err
	}
	if ipInfoCodec == nil {
		logger.WithField("fixed-ip", ip).Error(`Fixed-ip is not allocated`)
		return types.ErrFixedIPNotAllocated
	}
	if ipInfoCodec.IPInfo.Status.Match(types.IPStatusInUse) {
		logger.WithField("fixed-ip", ip).Error(`Fixed-ip in use`)
		return types.ErrIPInUse
	}

	ipInfoCodec.IPInfo.Status.Mark(types.IPStatusInUse)
	if ok, err = alloc.store.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Error("Update IPInfo error")
		return err
	} else if !ok {
		// update failed, the ip is modified by another container
		return types.ErrIPInUse
	}

	return nil
}

// UnassignFixedIP .
func (alloc fixedIPAllocator) UnassignFixedIP(ctx context.Context, ip types.IP) error {
	logger := alloc.logger(
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

	if ipInfoCodec, err = alloc.getFixedIP(ctx, ip, false); err != nil {
		return err
	}
	if ipInfoCodec == nil {
		logger.Error("IP is not allocated")
		return types.ErrFixedIPNotAllocated
	}

	if !ipInfoCodec.IPInfo.Status.Match(types.IPStatusInUse) {
		logger.Warn("FixedIP is already unassigned")
		return nil
	}

	ipInfoCodec.IPInfo.Status.Unmark(types.IPStatusInUse)
	if ok, err = alloc.store.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Errorf("Update IPInfo error")
		return err
	} else if !ok {
		return types.ErrIPInUse
	}

	return nil
}

// BorrowFixedIP .
func (alloc fixedIPAllocator) BorrowFixedIP(ctx context.Context, ip types.IP, container types.Container) error {
	logger := alloc.logger("BorrowFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	logger.WithField("ip", ip).WithField("container", container).Info("BorrowFixedIP")

	cnt := 0
	for cnt < retryMaxCount {
		codec, err := alloc.getFixedIP(ctx, ip, false)
		if err != nil {
			return err
		}
		attrs := codec.IPInfo.Attrs
		if attrs == nil {
			attrs = &types.IPAttributes{}
			codec.IPInfo.Attrs = attrs
		}
		attrs.Borrowers = append(attrs.Borrowers, container)
		if ok, err := alloc.store.UpdateElseGet(ctx, codec); err != nil {
			return err
		} else if ok {
			return nil
		}
		cnt++
	}
	return types.ErrMaxRetryCountExceeded
}

// ReturnFixedIP .
func (alloc fixedIPAllocator) ReturnFixedIP(ctx context.Context, ip types.IP, container types.Container) error {
	logger := alloc.logger("ReturnFixedIP")
	ctx = utils.WithEntry(ctx, logger)

	cnt := 0
	for cnt < retryMaxCount {
		codec, err := alloc.getFixedIP(ctx, ip, false)
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
		if ok, err := alloc.store.UpdateElseGet(ctx, codec); err != nil {
			return err
		} else if ok {
			return nil
		}
		cnt++
	}
	return types.ErrMaxRetryCountExceeded
}

// AllocFixedIP .
func (alloc fixedIPAllocator) AllocFixedIP(ctx context.Context, ip types.IP) error {
	ctx = alloc.context(ctx, "AllocFixedIP")

	// First check whether the ip is assigned as fixed ip
	var (
		ipInfoCodec *codecs.IPInfoCodec
		err         error
	)
	if ipInfoCodec, err = alloc.getFixedIP(ctx, ip, true); err != nil {
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

// UnallocFixedIP .
func (alloc fixedIPAllocator) UnallocFixedIP(ctx context.Context, ip types.IP) error {
	logger := alloc.logger(
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

	if ipInfoCodec, err = alloc.getFixedIP(ctx, ip, false); err != nil {
		return err
	}

	if ipInfoCodec == nil {
		logger.Warnf("IP is not allocated")
		return types.ErrFixedIPNotAllocated
	}

	ipInfo := ipInfoCodec.IPInfo
	if ipInfo.Status.Match(types.IPStatusInUse) {
		return types.ErrIPInUse
	}

	if ipInfo.Attrs != nil && len(ipInfo.Attrs.Borrowers) > 0 {
		logger.Info("IP still has borrower")
		return types.ErrFixedIPHasBorrower
	}

	// Lock the ip first
	ipInfo.Status.Mark(types.IPStatusInUse, types.IPStatusRetired)
	if ok, err = alloc.store.UpdateElseGet(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Errorf("Lock IPInfo failed")
	} else if !ok {
		return types.ErrIPInUse
	}

	// Now we remove the ipInfo
	if err = alloc.store.Delete(ctx, ipInfoCodec); err != nil {
		logger.WithError(err).Error("Delete IPInfo failed")
		// The
		return err
	}

	// Now we free the address
	if err = alloc.UnallocIP(ctx, ip); err != nil {
		logger.WithError(err).Error("Unalloc IP failed")
		return err
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
	if err = alloc.store.Put(ctx, ipInfoCodec); err != nil {
		if err := alloc.UnallocIP(ctx, ip.IP); err != nil {
			logger.WithError(err).Errorf("UnallocIP error")
		}
		return ip, err
	}
	return ip, nil
}

func (alloc fixedIPAllocator) getFixedIP(ctx context.Context, ip types.IP, alloIfAbsent bool) (*codecs.IPInfoCodec, error) {
	logger := utils.LogEntry(ctx)
	// First check whether the ip is assigned as fixed ip
	var (
		ipInfo           = types.IPInfo{Address: ip.Address, PoolID: ip.PoolID}
		ipInfoCodec      = &codecs.IPInfoCodec{IPInfo: &ipInfo}
		fixedIPNotExists = false
	)
	if err := alloc.store.Get(ctx, ipInfoCodec); err != nil {
		if !store.IsNotExists(err) {
			logger.WithError(err).Error("Get fixed-ip info error")
			return nil, err
		}
		fixedIPNotExists = true
	}
	if fixedIPNotExists {
		if !alloIfAbsent {
			return nil, nil
		}

		if err := alloc.AllocIP(ctx, ip); err != nil {
			logger.WithError(err).Error("Alloc IP error")
			return nil, err
		}
		if err := alloc.store.Put(ctx, ipInfoCodec); err != nil {
			logger.WithError(err).Error("Create FixedIPInfo error")
			return nil, err
		}
	}
	return ipInfoCodec, nil
}

func (alloc fixedIPAllocator) logger(method string) *log.Entry {
	return log.WithField("Receiver", "fixedIPAllocator").WithField("Method", method)
}

func (alloc fixedIPAllocator) context(ctx context.Context, method string) context.Context {
	return utils.WithEntry(ctx, alloc.logger(method))
}
