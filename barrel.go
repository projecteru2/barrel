package main

import (
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"

	minions "github.com/projecteru2/minions/lib"

	"github.com/docker/go-connections/sockets"
	dockerProxy "github.com/projecteru2/barrel/internal/proxy"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	workingDir    = "/run/barrel"
	proxyFilePath = workingDir + "/proxy.sock"
)

var (
	config            *apiconfig.CalicoAPIConfig
	calico            calicov3.Interface
	etcd              *etcdv3.Client
	debug             bool
	dockerdSocketPath string
)

func initialize() {
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Debug logging enabled")
	}

	var err error

	if config, err = apiconfig.LoadClientConfig(""); err != nil {
		log.Fatalln(err)
	}
	if calico, err = calicov3.New(*config); err != nil {
		log.Fatalln(err)
	}
	if etcd, err = minions.NewEtcdClient(strings.Split(config.Spec.EtcdConfig.EtcdEndpoints, ",")); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	app := &cli.App{
		Name:   "Barrel",
		Usage:  "Dockerd proxy for fixed IP",
		Action: run,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "debug or not",
				Destination: &debug,
				EnvVars:     []string{"BARREL_DEBUG"},
			},
			&cli.StringFlag{
				Name:        "dockerd-socket",
				Aliases:     []string{"ds"},
				Value:       "/var/run/docker.sock",
				Usage:       "dockerd socket",
				Destination: &dockerdSocketPath,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func run(c *cli.Context) error {
	initialize()

	var err error

	log.Printf(
		"Hello Barrel, dockerdSocketPath = %s",
		dockerdSocketPath,
	)
	if err = os.MkdirAll(workingDir, 0755); err != nil {
		return err
	}

	var group *user.Group
	if group, err = user.LookupGroup("docker"); err != nil {
		return err
	}
	log.Printf("the Gid of group docker is %s\n", group.Gid)

	var gid int64
	if gid, err = strconv.ParseInt(group.Gid, 10, 64); err != nil {
		return err
	}

	var listener net.Listener
	if listener, err = sockets.NewUnixSocket(proxyFilePath, int(gid)); err != nil {
		return err
	}

	config := dockerProxy.ProxyConfig{
		DockerdSocketPath: dockerdSocketPath,
		DialTimeout:       time.Duration(2) * time.Second,
	}

	return dockerProxy.NewProxy(config, etcd, calico).Start(listener)
}
