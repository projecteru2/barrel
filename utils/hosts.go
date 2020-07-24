package utils

import (
	"net"
	"strings"

	"github.com/docker/go-connections/sockets"
	"github.com/juju/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/types"
)

const (
	unixPrefix  = "unix://"
	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

// HostsParser .
type HostsParser struct {
	dockerGid   int64
	tlsCertFlag string
	tlsKeyFlag  string
}

// NewHostsParser .
func NewHostsParser(dockerGid int64, tlsCertFlag, tlsKeyFlag string) *HostsParser {
	return &HostsParser{
		dockerGid:   dockerGid,
		tlsCertFlag: tlsCertFlag,
		tlsKeyFlag:  tlsKeyFlag,
	}
}

// Parse .
func (h *HostsParser) Parse(hs []string) (hosts []types.Host, err error) {
	for _, value := range hs {
		var host types.Host
		if host, err = h.newHost(value); err != nil {
			return
		}
		hosts = append(hosts, host)
	}
	return
}

func (h *HostsParser) newHost(address string) (types.Host, error) {
	switch {
	case strings.HasPrefix(address, unixPrefix):
		return h.newUnixHost(strings.TrimPrefix(address, unixPrefix))
	case strings.HasPrefix(address, httpPrefix):
		return h.newHTTPHost(strings.TrimPrefix(address, httpPrefix))
	case strings.HasPrefix(address, httpsPrefix):
		return h.newHTTPSHost(strings.TrimPrefix(address, httpsPrefix))
	}
	return types.Host{}, errors.Errorf("unsupported protocol schema %s", address)
}

func (h *HostsParser) newUnixHost(address string) (host types.Host, err error) {
	if host.Listener, err = sockets.NewUnixSocket(address, int(h.dockerGid)); err != nil {
		return
	}
	return
}

func (h *HostsParser) newHTTPHost(address string) (host types.Host, err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}
	host.Listener = listener
	return
}

func (h *HostsParser) newHTTPSHost(address string) (host types.Host, err error) {
	if h.tlsCertFlag == "" || h.tlsKeyFlag == "" {
		return host, common.ErrCertAndKeyMissing
	}

	var listener net.Listener
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}
	host.Listener = listener
	host.Cert = h.tlsCertFlag
	host.Key = h.tlsKeyFlag
	return
}
