package ctr

import (
	// "encoding/json"
	// "os"

	// log "github.com/sirupsen/logrus"

	etcd "github.com/coreos/etcd/clientv3"
	dockerClient "github.com/docker/docker/client"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/clientv3"

	// "github.com/projectcalico/libcalico-go/lib/options"
	// cli "github.com/urfave/cli/v2"

	barrelEtcd "github.com/projecteru2/barrel/etcd"
	barrelStore "github.com/projecteru2/barrel/store"
	etcdStore "github.com/projecteru2/barrel/store/etcd"
	"github.com/projecteru2/barrel/vessel"
)

// Ctr .
type Ctr struct {
	apiConfig   *apiconfig.CalicoAPIConfig
	etcd        *etcd.Client
	calico      clientv3.Interface
	dockerCli   *dockerClient.Client
	store       barrelStore.Store
	ipAllocator vessel.FixedIPAllocator
	ipPool      vessel.FixedIPPool
	backend     bapi.Client
}

// InitCtr .
func InitCtr(ctr *Ctr, initFunc func(*Init)) error {
	ctrInit := Init{c: ctr}
	initFunc(&ctrInit)
	return ctrInit.init()
}

// Init .
type Init struct {
	c *Ctr
	InitUnits
}

// InitConfig .
func (c *Init) InitConfig() *InitUnit {
	return c.Declare(func() (err error) {
		if c.c.apiConfig != nil {
			return nil
		}
		c.c.apiConfig, err = apiconfig.LoadClientConfig("")
		return err
	})
}

// InitCalico .
func (c *Init) InitCalico() *InitUnit {
	return c.Require(func() (err error) {
		c.c.calico, err = clientv3.New(*c.c.apiConfig)
		return err
	}, c.InitConfig)
}

// InitDocker .
func (c *Init) InitDocker(dockerHost string, dockerVersion string) *InitUnit {
	return c.Declare(func() (err error) {
		c.c.dockerCli, err = dockerClient.NewClient(dockerHost, dockerVersion, nil, nil)
		return err
	})
}

// InitEtcd .
func (c *Init) InitEtcd() *InitUnit {
	return c.Require(func() (err error) {
		c.c.etcd, err = barrelEtcd.NewClient(c.c.apiConfig)
		return err
	}, c.InitConfig)
}

// InitStore .
func (c *Init) InitStore() *InitUnit {
	return c.Require(func() (err error) {
		c.c.store = etcdStore.NewEtcdStore(c.c.etcd)
		return nil
	}, c.InitEtcd)
}

// InitAllocator .
func (c *Init) InitAllocator(host string) *InitUnit {
	return c.Require(
		func() (err error) {
			c.c.ipAllocator = vessel.NewFixedIPAllocator(vessel.NewCalicoIPAllocator(
				c.c.calico, host,
			), c.c.store)
			return nil
		},
		c.InitCalico,
		c.InitStore,
	)
}

// InitPoolManager .
func (c *Init) InitPoolManager() *InitUnit {
	return c.Require(
		func() (err error) {
			c.c.ipPool = vessel.NewFixedIPPool(vessel.NewCalicoIPPool(
				c.c.calico,
			), c.c.store)
			return nil
		},
		c.InitCalico,
		c.InitStore,
	)
}

// InitCalicoBackend .
func (c *Init) InitCalicoBackend() *InitUnit {
	return c.Require(func() (err error) {
		c.c.backend, err = backend.NewClient(*c.c.apiConfig)
		return err
	}, c.InitConfig)
}
