package calicoplus

import (
	dockerClient "github.com/docker/docker/client"
	pluginIPAM "github.com/docker/go-plugins-helpers/ipam"
	pluginNetwork "github.com/docker/go-plugins-helpers/network"
	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/clientv3"
	calicoDriver "github.com/projecteru2/barrel/driver/calicoplus/calico"
)

const (
	ipamSuffix = "-ipam"
)

// NewDrivers .
func NewDrivers(
	driverName string,
	clientv3 clientv3.Interface,
	stor store.Store,
	dockerClient *dockerClient.Client,
) (driver.AddressManager, pluginNetwork.Driver) {
	ipam := &ipamDriver{
		driverName:   driverName,
		calicoIPAM:   calicoDriver.NewCalicoIPAM(clientv3),
		barrelEtcd:   stor,
		dockerClient: dockerClient,
	}
	return ipam, &networkDriver{
		calNetDriver: calicoDriver.NewNetworkDriver(clientv3, dockerClient),
		dockerCli:    dockerClient,
		ipam:         ipam,
	}
}

// RunNetworkPlugin .
func RunNetworkPlugin(driverName string, ipam pluginIPAM.Ipam, net pluginNetwork.Driver) error {
	errChannel := utils.NewWriteOnceChannel()

	networkHandler := pluginNetwork.NewHandler(net)
	ipamHandler := pluginIPAM.NewHandler(ipam)

	go func() {
		log.Infoln("calico-net has started.")
		err := networkHandler.ServeUnix(driverName, 0)
		log.Infoln("calico-net has stopped working.")
		errChannel.Send(err)
	}()

	go func() {
		log.Infoln("calico-ipam has started.")
		err := ipamHandler.ServeUnix(driverName+ipamSuffix, 0)
		log.Infoln("calico-ipam has stopped working.")
		errChannel.Send(err)
	}()

	return errChannel.Wait()
}
