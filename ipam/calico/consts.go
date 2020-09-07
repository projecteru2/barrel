package calico

import (
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	// PoolIDV4 .
	// Calico IPAM module does not allow selection of pools from which to allocate
	// IP addresses.  The pool ID, which has to be supplied in the libnetwork IPAM
	// API is therefore fixed.  We use different values for IPv4 and IPv6 so that
	// during allocation we know which IP version to use.
	PoolIDV4 = "CalicoPoolIPv4"
	// PoolIDV6 .
	PoolIDV6 = "CalicoPoolIPv6"
	// Calico IPAM module does not allow selection of pools from which to allocate
	// IP addresses.  The pool ID, which has to be supplied in the libnetwork IPAM
	// API is therefore fixed.  We use different values for IPv4 and IPv6 so that
	// during allocation we know which IP version to use.

	// CalicoLocalAddressSpace .
	CalicoLocalAddressSpace = "CalicoLocalAddressSpace"
	// CalicoGlobalAddressSpace .
	CalicoGlobalAddressSpace = "CalicoGlobalAddressSpace"
)

// IFPrefix .
var IFPrefix = "cali"

func init() { // nolint
	if os.Getenv("CALICO_LIBNETWORK_IFPREFIX") != "" {
		IFPrefix = os.Getenv("CALICO_LIBNETWORK_IFPREFIX")
		log.Infof("Updated CALICO_LIBNETWORK_IFPREFIX to %s", IFPrefix)
	}
}
