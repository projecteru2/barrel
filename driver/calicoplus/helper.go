package calicoplus

import (
	"fmt"
	"strings"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/juju/errors"
	caliconet "github.com/projectcalico/libcalico-go/lib/net"
)

const (
	fixedIPLabel = "fixed-ip"
)

func formatIPAddress(ip caliconet.IP) string {
	if ip.Version() == 4 {
		// IPv4 address
		return fmt.Sprintf("%v/%v", ip, "32")
	}
	// IPv6 address
	return fmt.Sprintf("%v/%v", ip, "128")
}

func checkOptions(options map[string]string) error {
	// Calico IPAM does not allow you to choose a gateway.
	if options["RequestAddressType"] == "com.docker.network.gateway" {
		err := errors.New("Calico IPAM does not support specifying a gateway")
		return err
	}
	return nil
}

func containerHasFixedIPLabel(container dockerTypes.Container) bool {
	value, hasFixedIPLabel := container.Labels[fixedIPLabel]
	return hasFixedIPLabel && strings.ToLower(value) != "false" && value != "0"
}
