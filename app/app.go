package app

import (
	"context"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"

	dockerClient "github.com/docker/docker/client"
	"github.com/juju/errors"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"
	log "github.com/sirupsen/logrus"

	"github.com/projecteru2/barrel/driver"
	calicoDriver "github.com/projecteru2/barrel/driver/calico"
	fixedIPDriver "github.com/projecteru2/barrel/driver/fixedip"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy/docker"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/store/etcd"
	"github.com/projecteru2/barrel/vessel"
)

// Application .
type Application struct {
	Hostname               string
	Mode                   string
	DockerDaemonUnixSocket string
	DockerAPIVersion       string
	Hosts                  []string
	DriverName             string
	IpamDriverName         string
	DialTimeout            time.Duration
	CertFile               string
	KeyFile                string
	ShutdownTimeout        time.Duration
	EnableCNMAgent         bool
}

// Run .
func (app Application) Run() error {
	switch app.Mode {
	case "default":
		log.Info("Running in default mode")
		return app.runMode(app.defaultMode)
	case "proxy-only":
		log.Info("Running in proxy only mode")
		return app.runMode(app.proxyOnlyMode)
	case "network-plugin-only":
		log.Info("Running in network plugin only mode")
		return app.runMode(app.networkPluginOnlyMode)
	default:
		return errors.New("Unrecognized barrel mode, support only [ --mode default | proxy-only | network-plugin-only ]")
	}
}

func (app Application) runMode(serviceFactory func() ([]service.Service, error)) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	defer signal.Stop(sigs)
	defer close(sigs)

	var (
		services []service.Service
		err      error
	)
	if services, err = serviceFactory(); err != nil {
		return err
	}
	return newStarter(services, app.ShutdownTimeout).start(sigs)
}

func (app Application) getAPIConfig() (*apiconfig.CalicoAPIConfig, error) {
	return apiconfig.LoadClientConfig("")
}

func (app Application) getDockerClient() (*dockerClient.Client, error) {
	return dockerClient.NewClient(app.DockerDaemonUnixSocket, app.DockerAPIVersion, nil, nil)
}

func (app Application) getEtcdClient(ctx context.Context, apiConfig *apiconfig.CalicoAPIConfig) (store.Store, error) {
	return etcd.NewClient(ctx, apiConfig)
}

func (app Application) getCalicoClient(apiConfig *apiconfig.CalicoAPIConfig) (calicov3.Interface, error) {
	return calicov3.New(*apiConfig)
}

func (app Application) defaultMode() ([]service.Service, error) {
	var (
		apiConfig *apiconfig.CalicoAPIConfig
		client    clientv3.Interface
		dockerCli *dockerClient.Client
		vess      vessel.Helper
		stor      store.Store
		agent     vessel.CNMAgent
		services  []service.Service
		gid       int
		err       error
	)
	if apiConfig, err = app.getAPIConfig(); err != nil {
		return nil, err
	}
	if client, err = app.getCalicoClient(apiConfig); err != nil {
		return nil, err
	}
	if dockerCli, err = app.getDockerClient(); err != nil {
		return nil, err
	}
	if stor, err = app.getEtcdClient(context.Background(), apiConfig); err != nil {
		return nil, err
	}
	if gid, err = getDockerGid(); err != nil {
		return nil, err
	}
	vess = vessel.NewHelper(vessel.NewVessel(app.Hostname, client, dockerCli, app.DriverName, stor), stor)
	if app.EnableCNMAgent {
		agent := vessel.NewAgent(vess, vessel.AgentConfig{})
		services = append(services, agent)
	}
	services = append(services, proxyService{
		Server: barrelHttp.NewServer(docker.NewHandler(app.DockerDaemonUnixSocket, app.DialTimeout, vess)),
		gid:    gid,
		tlsConfig: barrelHttp.TLSConfig{
			CertFile: app.CertFile,
			KeyFile:  app.KeyFile,
		},
		hosts: app.Hosts,
	},
		pluginService{
			ipam:   fixedIPDriver.NewIpam(vess.FixedIPAllocator()),
			driver: fixedIPDriver.NewDriver(client, dockerCli, agent, app.Hostname),
			server: driver.NewPluginServer(app.DriverName, app.IpamDriverName),
		})
	return services, nil
}

func (app Application) proxyOnlyMode() ([]service.Service, error) {
	var (
		gid int
		err error
	)
	if gid, err = getDockerGid(); err != nil {
		return nil, err
	}
	return []service.Service{
		proxyService{
			Server: barrelHttp.NewServer(docker.NewSimpleHandler(app.DockerDaemonUnixSocket, app.DialTimeout)),
			gid:    gid,
			tlsConfig: barrelHttp.TLSConfig{
				CertFile: app.CertFile,
				KeyFile:  app.KeyFile,
			},
			hosts: app.Hosts,
		},
	}, nil
}

// we will only launch calico plugin here, and fixed ip is not enabled
func (app Application) networkPluginOnlyMode() ([]service.Service, error) {
	var (
		apiConfig *apiconfig.CalicoAPIConfig
		client    clientv3.Interface
		dockerCli *dockerClient.Client
		allocator vessel.CalicoIPAllocator
		err       error
	)
	if apiConfig, err = app.getAPIConfig(); err != nil {
		return nil, err
	}
	if client, err = app.getCalicoClient(apiConfig); err != nil {
		return nil, err
	}
	if dockerCli, err = app.getDockerClient(); err != nil {
		return nil, err
	}
	allocator = vessel.NewIPPoolManager(client, dockerCli, app.DriverName, app.Hostname)
	return []service.Service{
		pluginService{
			ipam:   calicoDriver.NewIpam(allocator),
			driver: calicoDriver.NewDriver(client, dockerCli, app.Hostname),
			server: driver.NewPluginServer(app.DriverName, app.IpamDriverName),
		},
	}, nil
}

func getDockerGid() (int, error) {
	var (
		group     *user.Group
		err       error
		dockerGid int64
	)
	if group, err = user.LookupGroup("docker"); err != nil {
		return 0, err
	}
	log.Printf("The Gid of group docker is %s", group.Gid)
	if dockerGid, err = strconv.ParseInt(group.Gid, 10, 64); err != nil {
		return 0, err
	}
	return int(dockerGid), nil
}
