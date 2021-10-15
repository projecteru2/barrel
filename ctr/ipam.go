package ctr

import (
	"context"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/options"

	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/vessel/codecs"
)

// Assigned .
func (c *Ctr) Assigned(ctx context.Context, ip cnet.IP) (bool, error) {
	_, err := c.calico.IPAM().GetAssignmentAttributes(ctx, ip)
	if err != nil {
		if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// InspectFixedIP .
func (c *Ctr) InspectFixedIP(ctx context.Context, ip types.IP) (*types.IPInfo, bool, error) {
	codec := codecs.IPInfoCodec{
		IPInfo: &types.IPInfo{
			PoolID:  ip.PoolID,
			Address: ip.Address,
		},
	}
	if err := c.store.Get(ctx, &codec); err != nil {
		if err == store.ErrKVNotExists {
			return nil, false, nil
		}
		return nil, false, err
	}
	return codec.IPInfo, true, nil
}

// ListFixedIP .
func (c *Ctr) ListFixedIP(ctx context.Context, poolname string) ([]*types.IPInfo, error) {
	codec := codecs.IPInfoMultiGetCodec{}
	if err := c.store.GetMulti(ctx, &codec); err != nil {
		return nil, err
	}
	var ipInfos []*types.IPInfo
	for _, c := range codec.Codecs {
		ipInfos = append(ipInfos, c.IPInfo)
	}
	return ipInfos, nil
}

// UnassignFixedIP .
func (c *Ctr) UnassignFixedIP(ctx context.Context, ip types.IP, unalloc bool) error {
	if unalloc {
		return c.ipAllocator.UnallocFixedIP(ctx, ip)
	}
	if err := c.ipAllocator.UnassignFixedIP(
		ctx,
		ip,
	); err != nil {
		if err == types.ErrFixedIPNotAllocated && unalloc {
			return c.ipAllocator.UnallocIP(ctx, ip)
		}
		return err
	}
	return nil
}

// AssignFixedIP .
func (c *Ctr) AssignFixedIP(ctx context.Context, ip types.IP) error {
	return c.ipAllocator.AssignFixedIP(
		ctx,
		ip,
	)
}

// AssignIP .
func (c *Ctr) AssignIP(ctx context.Context, ip types.IP) error {
	return c.ipAllocator.AllocIP(
		ctx,
		ip,
	)
}

// ListBlocks .
func (c *Ctr) ListBlocks(ctx context.Context, opt ListBlockOpt) (result []*model.AllocationBlock, err error) {
	var (
		ipPool *v3.IPPool
		opts   = opt.ListInterface()
		blocks = []cnet.IPNet{}
	)
	if ipPool, err = opt.IPPool(ctx, c.calico.IPPools()); err != nil {
		return nil, err
	}

	datastoreObjs, err := c.backend.List(ctx, opts, "")

	if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
		// The block path does not exist yet.  This is OK - it means
		// there are no affine blocks.
		return result, nil
	} else if err != nil {
		log.Errorf("Error getting affine blocks: %v", err)
		return nil, err
	}

	// Iterate through and extract the block CIDRs.
	for _, o := range datastoreObjs.KVPairs {
		k := o.Key.(model.BlockKey)

		if ipPool == nil {
			blocks = append(blocks, k.CIDR)
			continue
		}

		// Add the block if no IP pools were specified, or if IP pools were specified
		// and the block falls within the given IP pools.
		var poolNet *cnet.IPNet
		_, poolNet, err = cnet.ParseCIDR(ipPool.Spec.CIDR)
		if err != nil {
			log.Errorf("Error parsing CIDR: %s from pool: %s %v", ipPool.Spec.CIDR, ipPool.Name, err)
			return nil, err
		}

		if poolNet.Contains(k.CIDR.IPNet.IP) {
			blocks = append(blocks, k.CIDR)
		}
	}

	var allocationBlocks []*model.AllocationBlock
	for _, cidr := range blocks {
		block, err := c.backend.Get(ctx, model.BlockKey{CIDR: cidr}, "")
		if err != nil {
			if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
				// the block doesn't exists
				continue
			}
			return nil, err
		}

		allocationBlock := block.Value.(*model.AllocationBlock)
		if opt.CheckAffinity(allocationBlock) {
			allocationBlocks = append(allocationBlocks, allocationBlock)
		}
	}

	return allocationBlocks, nil
}

// GetBlock .
func (c *Ctr) GetBlock(ctx context.Context, cidr cnet.IPNet, poolname string) (result *model.AllocationBlock, err error) {
	block, err := c.backend.Get(ctx, model.BlockKey{CIDR: cidr}, "")
	if err != nil {
		return nil, err
	}
	allocationBlock := block.Value.(*model.AllocationBlock)
	return allocationBlock, nil
}

// ReleaseEmptyBlock .
func (c *Ctr) ReleaseEmptyBlock(ctx context.Context, cidr cnet.IPNet, host string) (err error) {
	return c.calico.IPAM().ReleaseAffinity(ctx, cidr, host, true)
}

// ListHostBlockOnPoolOpt .
type ListHostBlockOnPoolOpt struct {
	Hostname string
	Poolname string
}

// ListHostBlockOpt .
type ListHostBlockOpt struct {
	Hostname string
}

// ListPoolBlockOpt .
type ListPoolBlockOpt struct {
	Poolname string
}

// ListInterface .
func (opt ListHostBlockOnPoolOpt) ListInterface() model.ListInterface {
	if opt.Hostname == "" {
		return model.BlockListOptions{IPVersion: 4}
	}
	return model.BlockAffinityListOptions{Host: opt.Hostname, IPVersion: 4}
}

// IPPool .
func (opt ListHostBlockOnPoolOpt) IPPool(ctx context.Context, ipPools clientv3.IPPoolInterface) (ipPool *v3.IPPool, err error) {
	if opt.Poolname == "" {
		return nil, nil
	}
	if ipPool, err = ipPools.Get(ctx, opt.Poolname, options.GetOptions{}); err != nil {
		log.Errorf("Invalid Pool - %v", opt.Poolname)
		return nil, err
	}
	return
}

// CheckAffinity .
func (opt ListHostBlockOnPoolOpt) CheckAffinity(block *model.AllocationBlock) bool {
	if opt.Hostname == "" {
		return true
	}
	return BlockHasAffinity(block, opt.Hostname)
}

// ListInterface .
func (opt ListHostBlockOpt) ListInterface() model.ListInterface {
	return model.BlockAffinityListOptions{Host: opt.Hostname, IPVersion: 4}
}

// IPPool .
func (opt ListHostBlockOpt) IPPool(ctx context.Context, ipPools clientv3.IPPoolInterface) (ipPool *v3.IPPool, err error) {
	return nil, nil
}

// CheckAffinity .
func (opt ListHostBlockOpt) CheckAffinity(block *model.AllocationBlock) bool {
	return BlockHasAffinity(block, opt.Hostname)
}

// ListInterface .
func (opt ListPoolBlockOpt) ListInterface() model.ListInterface {
	return model.BlockListOptions{IPVersion: 4}
}

// IPPool .
func (opt ListPoolBlockOpt) IPPool(ctx context.Context, ipPools clientv3.IPPoolInterface) (ipPool *v3.IPPool, err error) {
	if ipPool, err = ipPools.Get(ctx, opt.Poolname, options.GetOptions{}); err != nil {
		log.Errorf("Invalid Pool - %v", opt.Poolname)
		return nil, err
	}
	return
}

// CheckAffinity .
func (opt ListPoolBlockOpt) CheckAffinity(block *model.AllocationBlock) bool {
	return true
}

// ListBlockOpt .
type ListBlockOpt interface {
	ListInterface() model.ListInterface
	IPPool(context.Context, clientv3.IPPoolInterface) (*v3.IPPool, error)
	CheckAffinity(*model.AllocationBlock) bool
}
