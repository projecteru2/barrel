package calico

import (
	"context"
	"fmt"
	"net"

	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"
)

// Ipam .
type Ipam struct {
	vessel.CalicoIPAllocator
	utils.ObjectLogger
}

// NewIpam .
func NewIpam(ipAllocator vessel.CalicoIPAllocator) Ipam {
	return Ipam{
		CalicoIPAllocator: ipAllocator,
		ObjectLogger: utils.ObjectLogger{
			ObjectName: "CalicoIPIpam",
		},
	}
}

// GetCapabilities .
func (ipam Ipam) GetCapabilities() (*pluginIpam.CapabilitiesResponse, error) {
	resp := pluginIpam.CapabilitiesResponse{}
	return &resp, nil
}

// GetDefaultAddressSpaces .
func (ipam Ipam) GetDefaultAddressSpaces() (*pluginIpam.AddressSpacesResponse, error) {
	resp := &pluginIpam.AddressSpacesResponse{
		LocalDefaultAddressSpace:  CalicoLocalAddressSpace,
		GlobalDefaultAddressSpace: CalicoGlobalAddressSpace,
	}
	return resp, nil
}

// RequestPool .
func (ipam Ipam) RequestPool(request *pluginIpam.RequestPoolRequest) (*pluginIpam.RequestPoolResponse, error) {
	logger := ipam.Logger("RequestPool")

	// Calico IPAM does not allow you to request a SubPool.
	if request.SubPool != "" {
		err := errors.New(
			"Calico IPAM does not support sub pool configuration " +
				"on 'docker create network'. Calico IP Pools " +
				"should be configured first and IP assignment is " +
				"from those pre-configured pools.",
		)
		logger.Error(err)
		return nil, err
	}

	if len(request.Options) != 0 {
		err := errors.New("Arbitrary options are not supported")
		logger.Error(err)
		return nil, err
	}

	var (
		pool types.Pool
		err  error
	)

	// If a pool (subnet on the CLI) is specified, it must match one of the
	// preconfigured Calico pools.
	if request.Pool != "" {
		if pool, err = ipam.GetPoolByID(context.Background(), request.Pool); err != nil {
			logger.Errorf("request calico pool error, %v", err)
			return nil, err
		}
	} else {
		pool = ipam.GetDefaultPool(request.V6)
	}

	// We use static pool ID and CIDR. We don't need to signal the
	// The meta data includes a dummy gateway address. This prevents libnetwork
	// from requesting a gateway address from the pool since for a Calico
	// network our gateway is set to a special IP.
	resp := &pluginIpam.RequestPoolResponse{
		PoolID: pool.Name,
		Pool:   pool.CIDR,
		Data:   map[string]string{"com.docker.network.gateway": pool.Gateway},
	}
	return resp, nil
}

// ReleasePool .
func (ipam Ipam) ReleasePool(*pluginIpam.ReleasePoolRequest) error {
	return nil
}

// RequestAddress .
func (ipam Ipam) RequestAddress(request *pluginIpam.RequestAddressRequest) (*pluginIpam.RequestAddressResponse, error) {
	logger := ipam.Logger("RequestAddress")

	// Calico IPAM does not allow you to choose a gateway.
	if err := CheckOptions(request.Options); err != nil {
		logger.Errorf("check request options failed, %v", err)
		return nil, err
	}

	var (
		address string
		err     error
		ctx     = context.Background()
	)

	if request.Address == "" {
		var ipAddr types.IPAddress
		if ipAddr, err = ipam.AllocIPFromPool(ctx, request.PoolID); err != nil {
			return nil, err
		}
		address = ipAddr.Address
	} else if err = ipam.AllocIP(ctx, types.IP{PoolID: request.PoolID, Address: request.Address}); err != nil {
		return nil, err
	} else {
		address = request.Address
	}

	resp := &pluginIpam.RequestAddressResponse{
		// Return the IP as a CIDR.
		Address: IPAddressToCidr(address),
	}
	return resp, nil
}

// ReleaseAddress .
func (ipam Ipam) ReleaseAddress(request *pluginIpam.ReleaseAddressRequest) error {
	return ipam.UnallocIP(context.Background(), types.IP{PoolID: request.PoolID, Address: request.Address})
}

// IPv4ToCidr .
func IPv4ToCidr(ip string) string {
	return fmt.Sprintf("%s/%s", ip, "32")
}

// IPv6ToCidr .
func IPv6ToCidr(ip string) string {
	return fmt.Sprintf("%s/%s", ip, "128")
}

// IPAddressToCidr .
func IPAddressToCidr(ip string) string {
	if net.ParseIP(ip).To4() != nil {
		// IPv4 address
		return IPv4ToCidr(ip)
	}
	return IPv6ToCidr(ip)
}

// CaliNetIPToCidr .
func CaliNetIPToCidr(ip caliconet.IP) string {
	if ip.Version() == 4 {
		// IPv4 address
		return IPv4ToCidr(fmt.Sprintf("%v", ip))
	}
	return IPv6ToCidr(fmt.Sprintf("%v", ip))
}

// CheckOptions .
func CheckOptions(options map[string]string) error {
	// Calico IPAM does not allow you to choose a gateway.
	if options["RequestAddressType"] == "com.docker.network.gateway" {
		err := errors.New("Calico IPAM does not support specifying a gateway")
		return err
	}
	return nil
}
