package calico

import (
	"context"
	"net"

	"github.com/juju/errors"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	calicoipam "github.com/projectcalico/libcalico-go/lib/ipam"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/options"
	osutils "github.com/projectcalico/libnetwork-plugin/utils/os"
	"github.com/projecteru2/barrel/types"
	log "github.com/sirupsen/logrus"
)

// IPAMDriver .
type IPAMDriver struct {
	cliv3 clientv3.Interface
}

// NewCalicoIPAM .
func NewCalicoIPAM(cliv3 clientv3.Interface) *IPAMDriver {
	return &IPAMDriver{cliv3}
}

// AssignIP .
func (c IPAMDriver) AssignIP(address string) (caliconet.IP, error) {
	var err error

	var hostname string
	if hostname, err = osutils.GetHostname(); err != nil {
		return caliconet.IP{}, err
	}

	ip := net.ParseIP(address)
	// Docker allows the users to specify any address.
	// We'll return an error if the address isn't in a Calico pool, but we don't care which pool it's in
	// (i.e. it doesn't need to match the subnet from the docker network).
	log.Debugln("Reserving a specific address in Calico pools")
	ipArgs := calicoipam.AssignIPArgs{
		IP:       caliconet.IP{IP: ip},
		Hostname: hostname,
	}

	if err = c.cliv3.IPAM().AssignIP(context.Background(), ipArgs); err != nil {
		log.Errorf("IP assignment error, data: %+v\n", ipArgs)
		return caliconet.IP{}, err
	}
	return caliconet.IP{IP: ip}, nil
}

// AutoAssign .
func (c IPAMDriver) AutoAssign(poolName string) (caliconet.IP, error) {
	var err error

	// No address requested, so auto assign from our pools.
	log.Infof("Auto assigning IP from Calico pools, poolID = %s", poolName)

	var hostname string
	if hostname, err = osutils.GetHostname(); err != nil {
		return caliconet.IP{}, err
	}

	// If the poolID isn't the fixed one then find the pool to assign from.
	// poolV4 defaults to nil to assign from across all pools.
	var poolV4 []caliconet.IPNet

	var poolV6 []caliconet.IPNet
	var numIPv4, numIPv6 int
	switch poolName {
	case PoolIDV4:
		numIPv4 = 1
		numIPv6 = 0
	case PoolIDV6:
		numIPv4 = 0
		numIPv6 = 1
	default:
		var version int

		var ipPool *apiv3.IPPool
		if ipPool, err = c.GetIPPool(poolName); err != nil {
			log.Errorf("Invalid Pool - %v", poolName)
			return caliconet.IP{}, err
		}

		var ipNet *caliconet.IPNet
		if _, ipNet, err = caliconet.ParseCIDR(ipPool.Spec.CIDR); err != nil {
			log.Errorf("Invalid CIDR - %v", poolName)
			return caliconet.IP{}, err
		}

		version = ipNet.Version()
		if version == 4 {
			poolV4 = []caliconet.IPNet{{IPNet: ipNet.IPNet}}
			numIPv4 = 1
			log.Debugln("Using specific pool ", poolV4)
		} else if version == 6 {
			poolV6 = []caliconet.IPNet{{IPNet: ipNet.IPNet}}
			numIPv6 = 1
			log.Debugln("Using specific pool ", poolV6)
		}
	}

	// Auto assign an IP address.
	// IPv4/v6 pool will be nil if the docker network doesn't have a subnet associated with.
	// Otherwise, it will be set to the Calico pool to assign from.
	var IPsV4 []caliconet.IP
	var IPsV6 []caliconet.IP
	if IPsV4, IPsV6, err = c.cliv3.IPAM().AutoAssign(
		context.Background(),
		calicoipam.AutoAssignArgs{
			Num4:      numIPv4,
			Num6:      numIPv6,
			Hostname:  hostname,
			IPv4Pools: poolV4,
			IPv6Pools: poolV6,
		},
	); err != nil {
		log.Errorln("IP assignment error")
		return caliconet.IP{}, err
	}
	IPs := append(IPsV4, IPsV6...)

	// We should only have one IP address assigned at this point.
	if len(IPs) != 1 {
		return caliconet.IP{}, errors.Errorf("Unexpected number of assigned IP addresses. "+
			"A single address should be assigned. Got %v", IPs)
	}
	return IPs[0], nil
}

// GetIPPool .
func (c IPAMDriver) GetIPPool(poolName string) (*apiv3.IPPool, error) {
	return c.cliv3.IPPools().Get(context.Background(), poolName, options.GetOptions{})
}

// ReleaseIP .
func (c IPAMDriver) ReleaseIP(poolName string, address string) error {
	ip := caliconet.IP{IP: net.ParseIP(address)}
	// Unassign the address.  This handles the address already being unassigned
	// in which case it is a no-op.
	if _, err := c.cliv3.IPAM().ReleaseIPs(context.Background(), []caliconet.IP{ip}); err != nil {
		log.Errorf("IP releasing error, ip: %v", ip)
		return err
	}

	return nil
}

// IPPools .
func (c IPAMDriver) IPPools() (*apiv3.IPPoolList, error) {
	return c.cliv3.IPPools().List(context.Background(), options.ListOptions{})
}

// RequestPool .
func (c IPAMDriver) RequestPool(cidr string) (*types.Pool, error) {
	var (
		ipNet *caliconet.IPNet
		pools *apiv3.IPPoolList
		err   error
	)

	if _, ipNet, err = caliconet.ParseCIDR(cidr); err != nil {
		log.Errorf("Invalid CIDR: %s, %v", cidr, err)
		return nil, err
	}

	if pools, err = c.IPPools(); err != nil {
		log.Errorf("[CalicoDriver::RequestPool] Get pools error, %v", err)
		return nil, err
	}

	for _, p := range pools.Items {
		if p.Spec.CIDR == ipNet.String() {
			var gateway string
			if ipNet.Version() == 4 {
				gateway = defaultAddress
			} else {
				gateway = "::/0"
			}
			return &types.Pool{
				CIDR:    p.Spec.CIDR,
				Name:    p.Name,
				Gateway: gateway,
			}, nil
		}
	}

	return nil, errors.Errorf("The requested subnet(%s) didn't match any CIDR of a "+
		"configured Calico IP Pool.", cidr)
}

// RequestPools .
func (c IPAMDriver) RequestPools(cidrs []string) ([]types.Pool, error) {
	var (
		ipNets = make(map[string]*caliconet.IPNet)
		pools  *apiv3.IPPoolList
		result []types.Pool
		err    error
	)
	for _, cidr := range cidrs {
		var ipNet *caliconet.IPNet
		if _, ipNet, err = caliconet.ParseCIDR(cidr); err != nil {
			log.Errorf("Invalid CIDR: %s, %v", cidr, err)
			return nil, err
		}
		ipNets[ipNet.String()] = ipNet
	}
	if pools, err = c.IPPools(); err != nil {
		log.Errorf("[CalicoDriver::RequestPool] Get pools error, %v", err)
		return nil, err
	}
	for _, p := range pools.Items {
		if ipNet, ok := ipNets[p.Spec.CIDR]; ok {
			var gateway string
			if ipNet.Version() == 4 {
				gateway = defaultAddress
			} else {
				gateway = "::/0"
			}
			result = append(result, types.Pool{
				CIDR:    p.Spec.CIDR,
				Name:    p.Name,
				Gateway: gateway,
			})
		}
	}
	if len(result) == 0 {
		return nil, errors.Errorf("The requested subnets(%v) didn't match any CIDR of a "+
			"configured Calico IP Pool.", cidrs)
	}
	return result, nil
}

// RequestDefaultPool .
func (c IPAMDriver) RequestDefaultPool(v6 bool) *types.Pool {
	if v6 {
		// Default the poolID to the fixed value.
		return &types.Pool{
			Name:    PoolIDV6,
			CIDR:    "::/0",
			Gateway: "::/0",
		}
	}
	// Default the poolID to the fixed value.
	return &types.Pool{
		Name:    PoolIDV4,
		CIDR:    defaultAddress,
		Gateway: defaultAddress,
	}
}
