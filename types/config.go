package types

import (
	"regexp"
	"strings"
	"time"

	"github.com/juju/errors"
)

type Mode string

const (
	ModeDefault           Mode = "default"
	ModeProxyOnly         Mode = "proxy-only"
	ModeNetworkPluginOnly Mode = "network-plugin-only"
)

type Config struct {
	Hostname               string
	Mode                   Mode
	DockerDaemonUnixSocket string
	DockerAPIVersion       string
	Hosts                  []string
	DriverName             string
	IpamDriverName         string
	DialTimeout            time.Duration
	RequestTimeout         time.Duration
	CertFile               string
	KeyFile                string
	ShutdownTimeout        time.Duration
	EnableCNMAgent         bool
	ResourcePathRegexps    []*regexp.Regexp
	RecycleTimeout         time.Duration
}

func ParseMode(value string) (Mode, error) {
	value = strings.Trim(value, " ")
	if value == "" {
		return ModeDefault, nil
	}
	switch value {
	case string(ModeDefault):
		return ModeDefault, nil
	case string(ModeProxyOnly):
		return ModeProxyOnly, nil
	case string(ModeNetworkPluginOnly):
		return ModeNetworkPluginOnly, nil
	default:
		return "", errors.Errorf("%s is not a valid mode", value)
	}
}
