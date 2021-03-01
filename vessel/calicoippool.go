package vessel

import (
	"context"
	"net"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/juju/errors"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	calicoipam "github.com/projectcalico/libcalico-go/lib/ipam"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"

	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

const addressAny = "::/0"

// CalicoIPAllocator .
type CalicoIPAllocator interface {
	AllocIP(ctx context.Context, ip types.IP) error
	AllocIPFromPool(ctx context.Context, poolID string) (types.IPAddress, error)
	UnallocIP(ctx context.Context, ip types.IP) error
	GetPoolByID(ctx context.Context, poolID string) (types.Pool, error)
	GetPoolByCIDR(ctx context.Context, cidr string) (types.Pool, error)
	GetPoolsByCIDRS(ctx context.Context, cidr []string) ([]types.Pool, error)
	GetDefaultPool(ipv6 bool) types.Pool
	AllocIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error)
	GetPoolsByNetworkName(ctx context.Context, name string) ([]types.Pool, error)
}

type manager struct {
	cliv3 clientv3.Interface
	utils.LoggerFactory
	driverName string
	dockerCli  *dockerClient.Client
	hostname   string
}

// NewIPPoolManager .
func NewIPPoolManager(cliv3 clientv3.Interface, dockerCli *dockerClient.Client, driverName string, hostname string) CalicoIPAllocator {
	return manager{
		cliv3:         cliv3,
		dockerCli:     dockerCli,
		driverName:    driverName,
		LoggerFactory: utils.NewObjectLogger("networkAgentImpl"),
		hostname:      hostname,
	}
}

// AssignIP .
func (m manager) AllocIP(ctx context.Context, ip types.IP) error {
	var err error

	netIP := net.ParseIP(ip.Address)
	// Docker allows the users to specify any address.
	// We'll return an error if the address isn't in a Calico pool, but we don't care which pool it's in
	// (i.e. it doesn't need to match the subnet from the docker network).
	log.Debugln("Reserving a specific address in Calico pools")
	ipArgs := calicoipam.AssignIPArgs{
		IP:       caliconet.IP{IP: netIP},
		Hostname: m.hostname,
	}

	if err = m.cliv3.IPAM().AssignIP(ctx, ipArgs); err != nil {
		log.Errorf("IP assignment error, data: %+v\n", ipArgs)
		return err
	}
	return nil
}

// AutoAssign .
func (m manager) AllocIPFromPool(ctx context.Context, poolID string) (types.IPAddress, error) {
	var err error

	// No address requested, so auto assign from our pools.
	log.Infof("Auto assigning IP from Calico pools, poolID = %s", poolID)

	// If the poolID isn't the fixed one then find the pool to assign from.
	// poolV4 defaults to nil to assign from across all pools.
	var poolV4 []caliconet.IPNet

	var poolV6 []caliconet.IPNet
	var numIPv4, numIPv6 int
	switch poolID {
	case PoolIDV4:
		numIPv4 = 1
		numIPv6 = 0
	case PoolIDV6:
		numIPv4 = 0
		numIPv6 = 1
	default:
		var version int

		var ipPool *apiv3.IPPool
		if ipPool, err = m.cliv3.IPPools().Get(ctx, poolID, options.GetOptions{}); err != nil {
			log.Errorf("Invalid Pool - %v", poolID)
			return types.IPAddress{}, err
		}

		var ipNet *caliconet.IPNet
		if _, ipNet, err = caliconet.ParseCIDR(ipPool.Spec.CIDR); err != nil {
			log.Errorf("Invalid CIDR - %v", poolID)
			return types.IPAddress{}, err
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
	var IPsV4 []caliconet.IPNet
	var IPsV6 []caliconet.IPNet
	if IPsV4, IPsV6, err = m.cliv3.IPAM().AutoAssign(
		context.Background(),
		calicoipam.AutoAssignArgs{
			Num4:      numIPv4,
			Num6:      numIPv6,
			Hostname:  m.hostname,
			IPv4Pools: poolV4,
			IPv6Pools: poolV6,
		},
	); err != nil {
		log.Errorln("IP assignment error")
		return types.IPAddress{}, err
	}
	IPs := append(IPsV4, IPsV6...)

	// We should only have one IP address assigned at this point.
	if len(IPs) != 1 {
		return types.IPAddress{}, errors.Errorf("Unexpected number of assigned IP addresses. "+
			"A single address should be assigned. Got %v", IPs)
	}
	return types.IPAddress{
		IP:      types.IP{PoolID: poolID, Address: formatIP(IPs[0])},
		Version: IPs[0].Version(),
	}, nil
}

func formatIP(ip caliconet.IPNet) string {
	return ip.IPNet.IP.String()
}

// GetIPPool .
func (m manager) GetPoolByID(ctx context.Context, poolID string) (types.Pool, error) {
	var (
		p     *apiv3.IPPool
		ipNet *caliconet.IPNet
		err   error
	)
	if p, err = m.cliv3.IPPools().Get(ctx, poolID, options.GetOptions{}); err != nil {
		return types.Pool{}, err
	}
	if _, ipNet, err = caliconet.ParseCIDR(p.Spec.CIDR); err != nil {
		return types.Pool{}, err
	}
	var gateway string
	if ipNet.Version() == 4 {
		gateway = defaultAddress
	} else {
		gateway = addressAny
	}
	return types.Pool{
		CIDR:    p.Spec.CIDR,
		Name:    p.Name,
		Gateway: gateway,
	}, nil
}

// ReleaseIP .
func (m manager) UnallocIP(ctx context.Context, ip types.IP) error {
	calicoIP := caliconet.IP{IP: net.ParseIP(ip.Address)}
	// Unassign the address.  This handles the address already being unassigned
	// in which case it is a no-op.
	if _, err := m.cliv3.IPAM().ReleaseIPs(ctx, []caliconet.IP{calicoIP}); err != nil {
		log.Errorf("IP releasing error, ip: %v", calicoIP)
		return err
	}

	return nil
}

// IPPools .
func (m manager) IPPools(ctx context.Context) (*apiv3.IPPoolList, error) {
	return m.cliv3.IPPools().List(ctx, options.ListOptions{})
}

// RequestPool .
func (m manager) GetPoolByCIDR(ctx context.Context, cidr string) (types.Pool, error) {
	var (
		ipNet *caliconet.IPNet
		pools *apiv3.IPPoolList
		err   error
	)

	if _, ipNet, err = caliconet.ParseCIDR(cidr); err != nil {
		log.Errorf("Invalid CIDR: %s, %v", cidr, err)
		return types.Pool{}, err
	}

	if pools, err = m.IPPools(ctx); err != nil {
		log.Errorf("[CalicoDriver::RequestPool] Get pools error, %v", err)
		return types.Pool{}, err
	}

	for _, p := range pools.Items {
		if p.Spec.CIDR == ipNet.String() {
			var gateway string
			if ipNet.Version() == 4 {
				gateway = defaultAddress
			} else {
				gateway = addressAny
			}
			return types.Pool{
				CIDR:    p.Spec.CIDR,
				Name:    p.Name,
				Gateway: gateway,
			}, nil
		}
	}

	return types.Pool{}, errors.Errorf("The requested subnet(%s) didn't match any CIDR of a "+
		"configured Calico IP Pool.", cidr)
}

// RequestPools .
func (m manager) GetPoolsByCIDRS(ctx context.Context, cidrs []string) ([]types.Pool, error) {
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
	if pools, err = m.IPPools(ctx); err != nil {
		log.Errorf("[CalicoDriver::RequestPool] Get pools error, %v", err)
		return nil, err
	}
	for _, p := range pools.Items {
		if ipNet, ok := ipNets[p.Spec.CIDR]; ok {
			var gateway string
			if ipNet.Version() == 4 {
				gateway = defaultAddress
			} else {
				gateway = addressAny
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
func (m manager) GetDefaultPool(v6 bool) types.Pool {
	if v6 {
		// Default the poolID to the fixed value.
		return types.Pool{
			Name:    PoolIDV6,
			CIDR:    addressAny,
			Gateway: addressAny,
		}
	}
	// Default the poolID to the fixed value.
	return types.Pool{
		Name:    PoolIDV4,
		CIDR:    defaultAddress,
		Gateway: defaultAddress,
	}
}

// ReserveAddressFromPools .
func (m manager) AllocIPFromPools(ctx context.Context, pools []types.Pool) (types.IPAddress, error) {
	logger := m.Logger("AssignIPFromPools")

	var (
		ip  types.IPAddress
		err error
	)
	if len(pools) == 1 {
		return m.AllocIPFromPool(ctx, pools[0].Name)
	}

	var poolNames []string
	for _, pool := range pools {
		if ip, err = m.AllocIPFromPool(ctx, pool.Name); err != nil {
			poolNames = append(poolNames, pool.Name)
			logger.Errorf("AutoAssign from %s error, %v", pool.Name, err)
			continue
		}
		return ip, nil
	}
	return types.IPAddress{}, errors.Errorf("AutoAssign from %v failed", poolNames)
}

func included(pools []types.Pool, poolID string) bool {
	for _, pool := range pools {
		if pool.Name == poolID {
			return true
		}
	}
	return false
}

// GetIPPoolsByNetworkName .
func (m manager) GetPoolsByNetworkName(ctx context.Context, name string) ([]types.Pool, error) {
	var (
		network dockerTypes.NetworkResource
		err     error
	)
	if network, err = m.dockerCli.NetworkInspect(ctx, name, dockerTypes.NetworkInspectOptions{}); err != nil {
		return nil, err
	}
	if network.Driver != m.driverName {
		return nil, types.ErrUnsupervisedNetwork
	}
	if len(network.IPAM.Config) == 0 {
		return nil, types.ErrConfiguredPoolUnfound
	}
	var cidrs []string
	for _, config := range network.IPAM.Config {
		cidrs = append(cidrs, config.Subnet)
	}
	return m.GetPoolsByCIDRS(ctx, cidrs)
}
