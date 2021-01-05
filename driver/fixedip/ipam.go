package fixedip

import (
	"context"

	calicoDriver "github.com/projecteru2/barrel/driver/calico"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"

	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
)

// Ipam .
type Ipam struct {
	calicoDriver.Ipam
	utils.ObjectLogger
	vessel.FixedIPAllocator
}

// NewIpam .
func NewIpam(
	allocator vessel.FixedIPAllocator,
) pluginIpam.Ipam {
	return Ipam{
		Ipam: calicoDriver.NewIpam(allocator),
		ObjectLogger: utils.ObjectLogger{
			ObjectName: "FixedIPIpam",
		},
		FixedIPAllocator: allocator,
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

	if err := ipam.AssignFixedIP(
		context.Background(),
		types.IP{
			PoolID:  request.PoolID,
			Address: request.Address,
		},
	); err != nil {
		if err == types.ErrFixedIPNotAllocated {
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
	if err := ipam.UnassignFixedIP(
		context.Background(),
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
