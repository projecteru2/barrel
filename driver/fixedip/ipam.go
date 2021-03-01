package fixedip

import (
	"context"
	"time"

	calicoDriver "github.com/projecteru2/barrel/driver/calico"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"

	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
)

// Ipam .
type Ipam struct {
	calicoDriver.Ipam
	utils.LoggerFactory
	vessel.FixedIPAllocator
	requestTimeout time.Duration
}

// NewIpam .
func NewIpam(
	allocator vessel.FixedIPAllocator,
	requestTimeout time.Duration,
) pluginIpam.Ipam {
	return Ipam{
		Ipam:             calicoDriver.NewIpam(allocator, requestTimeout),
		LoggerFactory:    utils.NewObjectLogger("FixedIPIpam"),
		FixedIPAllocator: allocator,
		requestTimeout:   requestTimeout,
	}
}

// RequestAddress .
func (ipam Ipam) RequestAddress(request *pluginIpam.RequestAddressRequest) (*pluginIpam.RequestAddressResponse, error) {
	logger := ipam.Logger("RequestAddress")

	// Calico IPAM does not allow you to choose a gateway.
	if err := calicoDriver.CheckOptions(request.Options); err != nil {
		logger.Errorf("check request options failed, %v", err)
		return nil, err
	}

	if request.Address == "" {
		return ipam.Ipam.RequestAddress(request)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ipam.requestTimeout)
	defer cancel()
	if err := ipam.AssignFixedIP(
		ctx,
		types.IP{
			PoolID:  request.PoolID,
			Address: request.Address,
		},
	); err != nil {
		if err == types.ErrFixedIPNotAllocated {
			logger.Debug("FixedIPNotAllocated")
			return ipam.Ipam.RequestAddress(request)
		}
		return nil, err
	}
	return &pluginIpam.RequestAddressResponse{
		// Return the IP as a CIDR.
		Address: calicoDriver.IPAddressToCidr(request.Address),
	}, nil
}

// ReleaseAddress .
func (ipam Ipam) ReleaseAddress(request *pluginIpam.ReleaseAddressRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), ipam.requestTimeout)
	defer cancel()
	if err := ipam.UnassignFixedIP(
		ctx,
		types.IP{
			PoolID:  request.PoolID,
			Address: request.Address,
		},
	); err != nil {
		if err == types.ErrFixedIPNotAllocated {
			return ipam.Ipam.ReleaseAddress(request)
		}
		return err
	}

	return nil
}
