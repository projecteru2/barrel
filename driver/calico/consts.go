package calico

import (
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	// CalicoLocalAddressSpace .
	CalicoLocalAddressSpace = "CalicoLocalAddressSpace"
	// CalicoGlobalAddressSpace .
	CalicoGlobalAddressSpace = "CalicoGlobalAddressSpace"

	dockerLabelPrefix      = "org.projectcalico.label."
	labelPollTimeoutEnvKey = "CALICO_LIBNETWORK_LABEL_POLL_TIMEOUT"
	createProfilesEnvKey   = "CALICO_LIBNETWORK_CREATE_PROFILES"
	labelEndpointsEnvKey   = "CALICO_LIBNETWORK_LABEL_ENDPOINTS"
	vethMTUEnvKey          = "CALICO_LIBNETWORK_VETH_MTU"
	namespaceEnvKey        = "CALICO_LIBNETWORK_NAMESPACE"
)

// IFPrefix .
var IFPrefix = "cali"

func init() { // nolint
	if os.Getenv("CALICO_LIBNETWORK_IFPREFIX") != "" {
		IFPrefix = os.Getenv("CALICO_LIBNETWORK_IFPREFIX")
		log.Infof("Updated CALICO_LIBNETWORK_IFPREFIX to %s", IFPrefix)
	}
}
