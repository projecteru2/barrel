package main

import (
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"

	minions "github.com/projecteru2/minions/lib"

	"github.com/docker/go-connections/sockets"
	"github.com/projecteru2/barrel/internal/proxy"
	dockerProxy "github.com/projecteru2/barrel/internal/proxy"
	"github.com/projecteru2/barrel/internal/utils"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

var (
	config            *apiconfig.CalicoAPIConfig
	calico            calicov3.Interface
	etcd              *etcdv3.Client
	debug             bool
	dockerdSocketPath string
	hostsFlag         string
	tlsCertFlag       string
	tlsKeyFlag        string
	dockerGid         int64
	ipPoolNames       string
)

func initialize() {
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Debug logging enabled")
	}
	utils.Initialize(256, debug)
	proxy.Initialize(debug)

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
				Aliases:     []string{"D"},
				Value:       "/var/run/docker.sock",
				Usage:       "dockerd socket",
				Destination: &dockerdSocketPath,
				EnvVars:     []string{"DOCKERD_SOCKET_PATH"},
			},
			&cli.StringFlag{
				Name:        "hosts",
				Aliases:     []string{"H"},
				Value:       "unix:///var/run/barrel.sock",
				Usage:       "hosts",
				Destination: &hostsFlag,
				EnvVars:     []string{"BARREL_HOSTS"},
			},
			&cli.StringFlag{
				Name:        "tls-cert",
				Aliases:     []string{"TC"},
				Usage:       "tls-cert-file-path",
				Destination: &tlsCertFlag,
				EnvVars:     []string{"BARREL_TLS_CERT_FILE_PATH"},
			},
			&cli.StringFlag{
				Name:        "tls-key",
				Aliases:     []string{"TK"},
				Usage:       "tls-key-file-path",
				Destination: &tlsKeyFlag,
				EnvVars:     []string{"BARREL_TLS_KEY_FILE_PATH"},
			},
			&cli.StringFlag{
				Name:        "ip-pools",
				Aliases:     []string{"P"},
				Usage:       "ip pool names",
				Destination: &ipPoolNames,
				EnvVars:     []string{"MINIONS_IP_POOL_NAMES"},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func run(c *cli.Context) (err error) {
	initialize()

	log.Printf(
		"Hello Barrel, dockerdSocketPath = %s",
		dockerdSocketPath,
	)

	var group *user.Group
	if group, err = user.LookupGroup("docker"); err != nil {
		return
	}
	log.Printf("the Gid of group docker is %s\n", group.Gid)

	if dockerGid, err = strconv.ParseInt(group.Gid, 10, 64); err != nil {
		return
	}

	config := dockerProxy.Config{
		DockerdSocketPath: dockerdSocketPath,
		DialTimeout:       time.Duration(2) * time.Second,
		IPPoolNames:       strings.Split(ipPoolNames, ","),
	}

	var hosts []proxy.Host
	if hosts, err = parseHosts(hostsFlag); err != nil {
		return
	}
	return dockerProxy.NewProxy(config, etcd, calico).Start(hosts...)
}

const (
	unixPrefix  = "unix://"
	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

func parseHosts(hostsArgs string) (hosts []proxy.Host, err error) {
	for _, value := range strings.Split(hostsArgs, ",") {
		var host proxy.Host
		if host, err = newHost(value); err != nil {
			return
		}
		hosts = append(hosts, host)
	}
	return
}

func newHost(address string) (proxy.Host, error) {
	if strings.HasPrefix(address, unixPrefix) {
		return newUnixHost(strings.TrimPrefix(address, unixPrefix))
	} else if strings.HasPrefix(address, httpPrefix) {
		return newHTTPHost(strings.TrimPrefix(address, httpPrefix))
	} else if strings.HasPrefix(address, httpsPrefix) {
		return newHTTPSHost(strings.TrimPrefix(address, httpsPrefix))
	}
	return proxy.Host{}, errors.Errorf("unsupported protocol schema %s", address)
}

func newUnixHost(address string) (host proxy.Host, err error) {
	if host.Listener, err = sockets.NewUnixSocket(address, int(dockerGid)); err != nil {
		return
	}
	return
}

func newHTTPHost(address string) (host proxy.Host, err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}
	host.Listener = listener
	return
}

func newHTTPSHost(address string) (host proxy.Host, err error) {
	if tlsCertFlag == "" || tlsKeyFlag == "" {
		err = errors.New("can't create https host without cert and key")
		return
	}

	var listener net.Listener
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}
	host.Listener = listener
	host.Cert = tlsCertFlag
	host.Key = tlsKeyFlag
	return
}
