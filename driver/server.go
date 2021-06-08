package driver

import (
	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
	pluginNetwork "github.com/docker/go-plugins-helpers/network"
	log "github.com/sirupsen/logrus"
)

const (
	// IpamSuffix .
	IpamSuffix = "-ipam"
	// DriverName .
	DriverName = "calico"
)

// PluginServer .
type PluginServer interface {
	ServeIpam(pluginIpam.Ipam) error
	ServeNetwork(pluginNetwork.Driver) error
}

type pluginServer struct {
	ipamDriverName string
	driverName     string
}

// NewPluginServer .
func NewPluginServer(driverName string, ipamDriverName string) PluginServer {
	return pluginServer{
		ipamDriverName: ipamDriverName,
		driverName:     driverName,
	}
}

func (s pluginServer) ServeIpam(ipam pluginIpam.Ipam) error {
	log.Infoln("start ipam.")
	if err := pluginIpam.NewHandler(ipamWrapper{ipam}).ServeUnix(s.ipamDriverName, 0); err != nil {
		log.WithError(err).Error("ipam has stopped working.")
		return err
	}
	log.Info("ipam stopped.")
	return nil
}

func (s pluginServer) ServeNetwork(driver pluginNetwork.Driver) error {
	log.Infoln("start net driver.")
	if err := pluginNetwork.NewHandler(driverWrapper{driver}).ServeUnix(s.driverName, 0); err != nil {
		log.WithError(err).Error("net driver has stopped working.")
		return err
	}
	log.Info("net driver stopped.")
	return nil
}
