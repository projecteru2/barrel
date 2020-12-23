package utils

import (
	"net"

	"github.com/juju/errors"
)

// Belongs .
func Belongs(ipAddr string, subnetCidr string) (bool, error) {
	var (
		ip     net.IP
		subnet *net.IPNet
		err    error
	)
	if _, subnet, err = net.ParseCIDR(subnetCidr); err != nil {
		return false, err
	}
	if ip = net.ParseIP(ipAddr); ip == nil {
		return false, errors.Errorf("Invalid IP Address: %s", ipAddr)
	}
	return subnet.Contains(ip), nil
}
